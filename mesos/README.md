# Deis Mesos Subsystem

The Mesos subsystem for Deis includes containers required to support
scheduling Deis application containers using Marathon.

## Containers

The Mesos meta-component is comprised of four containers:

* [mesos-zk](https://index.docker.io/u/deis/mesos-zk/) - Zookeeper cluster for use by Mesos
* [mesos-master](https://index.docker.io/u/deis/mesos-master/) - Mesos master used to coordinate cluster resources
* [mesos-marathon](https://index.docker.io/u/deis/mesos-marathon/) - Marathon framework used for scheduling
* [mesos-slave](https://index.docker.io/u/deis/mesos-slave/) - Mesos slave used to dispatch work on the data plane

Master and slave images are based on [mesos-base](https://github.com/deis/deis/tree/master/mesos/base) image,
which is a containerized Mesos environment.

## Usage

Please consult the [Makefile](Makefile) for current instructions on how to build, test, push,
install, and start **deis/mesos**.

## License

© 2014 OpDemand LLC

Licensed under the Apache License, Version 2.0 (the "License"); you may
not use this file except in compliance with the License. You may obtain
a copy of the License at <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
