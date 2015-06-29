:title: Choosing a Scheduler
:description: How to choose a scheduler backend for Deis.


.. _choosing_a_scheduler:

Choosing a Scheduler
====================

The :ref:`scheduler` creates, starts, stops, and destroys each :ref:`container`
of your app. For example, a command such as ``deis scale web=3`` tells the
scheduler to run three containers from the Docker image for your app.

Deis defaults to using the `Fleet Scheduler`_. A tech preview of the `Swarm Scheduler`_
and `Mesos with Marathon framework`_ is available for testing. Work is ongoing on `Kubernetes`_-based
scheduler with the intent to test those alternatives in future releases of Deis. Keep watching issue `deis-kubernetes issue 3850`_


Settings set by scheduler
-------------------------

The following etcd keys are set by the scheduler module of the controller component.

Some keys will exist only if a particular ``schedulerModule`` backend is enabled.

=============================            ================================================
setting                                  description
=============================            ================================================
/deis/scheduler/swarm/host               the swarm manager's host IP address
/deis/scheduler/swarm/node               used to identify other nodes in the cluster
/deis/scheduler/mesos/marathon           used to identify Marathon framework's host IP address
=============================            ================================================


Settings used by scheduler
--------------------------

The following etcd keys are used by the scheduler module of the controller component.

====================================      ===============================================
setting                                   description
====================================      ===============================================
/deis/controller/schedulerModule          scheduler backend, either "fleet" or "swarm" or
                                          mesos_marathon (default: "fleet")
====================================      ===============================================


Fleet Scheduler
---------------

`fleet`_ is a scheduling backend included with CoreOS:

    fleet ties together systemd and etcd into a distributed init system. Think of
    it as an extension of systemd that operates at the cluster level instead of the
    machine level. This project is very low level and is designed as a foundation
    for higher order orchestration.

``fleetd`` is already running on the machines provisioned for Deis: no additional
configuration is needed. Commands such as ``deis ps:restart web.1`` or
``deis scale cmd=10`` will use `fleet`_ by default to manage app containers.

To use the Fleet Scheduler backend explicitly, set the controller's
``schedulerModule`` to "fleet":

.. code-block:: console

    $ deisctl config controller set schedulerModule=fleet


Swarm Scheduler
---------------

.. important::

    The Swarm Scheduler is a technology preview and is not recommended for
    production use.

`swarm`_ is a scheduling backend for Docker:

    Docker Swarm is native clustering for Docker. It turns a pool of Docker hosts
    into a single, virtual host.

..

    Swarm serves the standard Docker API, so any tool which already communicates
    with a Docker daemon can use Swarm to transparently scale to multiple hosts...

Deis includes an enhanced version of swarm v0.2.0 with node failover and optimized
locking on container creation. The Swarm Scheduler uses a `soft affinity`_ filter
to spread app containers out among available machines.

Swarm requires the Docker Remote API to be available at TCP port 2375. If you are
upgrading an earlier installation of Deis, please refer to the CoreOS documentation
to `enable the remote API`_.

.. note::

    **Known Issues**

    - It is not yet possible to change the default affinity filter.
    - If swarm can't create all the containers requested during scale, deis rolls back the scale operation.
    - App containers will not be rescheduled if deis-registry is unavailable.

To test the Swarm Scheduler backend, first install and start the swarm components:

.. code-block:: console

    $ deisctl install swarm && deisctl start swarm

Then set the controller's ``schedulerModule`` to "swarm":

.. code-block:: console

    $ deisctl config controller set schedulerModule=swarm

The Swarm Scheduler is now active. Commands such as ``deis destroy`` or
``deis scale web=9`` will use `swarm`_ to manage app containers.

To monitor Swarm Scheduler operations, watch the logs of the swarm-manager
component, or spy on Docker events directly on the swarm-manager machine:

.. code-block:: console

    $ deisctl journal swarm-manager
    $ docker -H 172.17.8.102:2395 events
    2015-04-30T17:31 172.17.8.100:5000/hungry-variable:v5: (from  node:deis-01) pull
    2015-04-30T17:31 172.17.8.100:5000/hungry-variable:v5: (from  node:deis-02) pull
    2015-04-30T17:31 02a570: (from 172.17.8.100:5000/hungry-variable:v5 node:deis-01) create
    2015-04-30T17:31 02a570: (from 172.17.8.100:5000/hungry-variable:v5 node:deis-01) start
    2015-04-30T17:31 61e59c: (from 172.17.8.100:5000/hungry-variable:v5 node:deis-02) create
    2015-04-30T17:31 61e59c: (from 172.17.8.100:5000/hungry-variable:v5 node:deis-02) start


Mesos with Marathon framework
-----------------------------

.. important::

    The Mesos with Marathon framework Scheduler is a technology preview and is not recommended for
    production use.

`Mesos`_ is a distributed system kernel:

    Mesos provides APIs for resource management and scheduling. A framework interacts with Mesos master
    and schedules and task. A Zookeeper cluster elects Mesos master node. Mesos slaves are installed on
    each node and they communicate to master with available resources.


..

`Marathon`_ is a Mesos_ framework for long running applications:

    Marathon provides a Paas like feel for long running applications and features like high-availablilty, host constraints,
    service discovery, load balancing and REST API to control your Apps.

..


Deis uses Marathon framework to schedule containers. As Marathon is a framework for long running jobs, we are using fleet to
run batch processing jobs. deisctl installs standlone Mesos cluster. If you want to install HA mesos cluster please follow
`aledbf-mesos`_ and set edcd key /deis/scheduler/mesos/marathon to any Marathon node IP address, if the request is received
by a non-master Marathon then the request is proxied to master Marathon node.

To test the Marathon Scheduler backend, first install and start the mesos components:

.. code-block:: console

    $ deisctl install mesos && deisctl start mesos

Then set the controller's ``schedulerModule`` to "mesos_marathon":

.. code-block:: console

    $ deisctl config controller set schedulerModule=mesos_marathon

The Marathon framework is now active. Commands such as ``deis destroy`` or
``deis scale web=9`` will use `Marathon`_ to manage app containers.

Deis starts Marathon on port 8180. You can manage the apps through Marathon UI which is accessible at http://<Marathon-node-IP>:8180

.. note::

    **Known Issues**

    - deisctl install a standalone mesos cluster. As deis doesn't support metadata tags for fleet.
      keep watching `dynamic metadata fleet PR 1077`_.
    - App containers will not be rescheduled if deis-registry is unavailable.
    - Marathon starts on port 8180.
    - Deis is not yet using docker container API of Marathon for creating containers.
    - While using CPU Shares its the integer value of number of Cpus and Memory limits unit should be in MB.
    

.. _Kubernetes: http://kubernetes.io/
.. _Mesos: http://mesos.apache.org
.. _Marathon: https://github.com/mesosphere/marathon#marathon-
.. _fleet: https://github.com/coreos/fleet#fleet---a-distributed-init-system
.. _swarm: https://github.com/docker/swarm#swarm-a-docker-native-clustering-system
.. _`soft affinity`: https://docs.docker.com/swarm/scheduler/filter/#soft-affinitiesconstraints
.. _`enable the remote API`: https://coreos.com/docs/launching-containers/building/customizing-docker/
.. _`deis-kubernetes issue 3850`: https://github.com/deis/deis/issues/3850
.. _`dynamic metadata fleet PR 1077`: https://github.com/coreos/fleet/pull/1077
.. _`aledbf-mesos`: https://github.com/aledbf/coreos-mesos-zookeeper
