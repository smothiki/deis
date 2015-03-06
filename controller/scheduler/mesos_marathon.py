import cStringIO
import base64
import copy
import json
import httplib
import logging
import paramiko
import socket
import re
import time

import marathon as marathon_api
from .fleet import FleetHTTPClient, RUN_TEMPLATE

# turn down standard marathon logging
marathon_api.log.setLevel(logging.CRITICAL)

MATCH = re.compile(
    '(?P<app>[a-z0-9-]+)_?(?P<version>v[0-9]+)?\.?(?P<c_type>[a-z-_]+)?.(?P<c_num>[0-9]+)')
RETRIES = 3


class MarathonHTTPClient(object):

    def __init__(self, target, auth, options, pkey):
        self.target = target
        self.auth = auth
        self.options = options
        self.pkey = pkey
        # use an instance-level connection object
        self.conn = marathon_api.MarathonClient(self.target)
        self.fleet = FleetHTTPClient('/var/run/fleet.sock', auth, options, pkey)

    # helpers

    def _app_id(self, name):
        return name.replace('_', '.')

    # container api

    def create(self, name, image, command='', **kwargs):
        """Create a container"""
        app_id = self._app_id(name)
        app = {"id": app_id,
               "args": command.split(' '),
               "container": {
                    "type": "DOCKER",
                    "docker": {
                        "image": self.options['registry']+"/"+image,
                        "network": "BRIDGE",
                        "portMappings": [ {
                            # FIXME: inject dynamic container ports
                            "containerPort": 5000,
                            "hostPort": 0,
                            "servicePort": 0,
                            "protocol": "tcp"
                    }]},
                },
                "cpus": 0.5,
                "env": {"DEIS_ID": name},
                "mem": 512.0,
                "ports": [0]
        }
        model = marathon_api.models.MarathonApp(**app)
        self._create_container(name, image, command, model, **kwargs)

    def _create_container(self, name, image, command, model, **kwargs):
        app_id = self._app_id(name)

        for _ in range(RETRIES):

            # create the app
            try:
                if not self.conn.create_app(app_id, model):
                    raise RuntimeError('marathon refused to create container')
            except marathon_api.exceptions.MarathonHttpError as e:

                # if app is locked, destroy old version and try again
                if e.status_code == 409:
                    self._destroy_container(name)
                    time.sleep(2)
                    continue
                raise

            # test for app existence
            try:
                self.conn.get_app(app_id)
                break
            except marathon_api.exceptions.MarathonHttpError as e:
                if e.status_code != 404:
                    raise

            # prepare to reset the app
            self._destroy_container(name)
            time.sleep(5)

            # create a fake app to reset
            reset = {"id": app_id, "args": ["sleep", "1"], "cpus": 0.1, "mem": 32,}
            reset_app = marathon_api.models.MarathonApp(**reset)
            if not self.conn.create_app(app_id, reset_app):
                raise RuntimeError('marathon refused to reset container')
            self._destroy_container(name)

            time.sleep(2)
        else:
            raise RuntimeError('marathon failed to create container')

    def start(self, name):
        """Start a container"""
        self._wait_for_container(name)

    def _wait_for_container(self, name):
        app_id = self._app_id(name)

        # wait up to 20 min for container to start
        for _ in range(1200):
            try:
                app = self.conn.get_app(app_id)
            except marathon_api.exceptions.MarathonHttpError as e:
                if e.status_code == 404:
                    return RuntimeError('marathon lost track of container')
                raise
            if app.tasks_running == 1:
                return app
            time.sleep(1)
        else:
            raise RuntimeError('marathon timeout on container start')

    def stop(self, name):
        """Stop a container"""
        raise NotImplementedError

    def destroy(self, name):
        """Destroy a container"""
        try:
            self._destroy_container(name)
        except:
            pass

    def _destroy_container(self, name):
        app_id = self._app_id(name)

        # bail early if app is already gone
        try:
            app = self.conn.get_app(app_id)
        except marathon_api.exceptions.MarathonHttpError as e:
            if e.status_code == 404:
                return

        try:
            # destroying the app until it actually goes away
            while self.conn.get_app(app_id):

                # scale down app instances
                self.conn.scale_app(app_id, instances=0, force=True)

                # wait 30s for scale down
                for _ in range(30):
                    app = self.conn.get_app(app_id)
                    if app.instances == 0:
                        break
                    time.sleep(1)

                # destroy the app and wait
                self.conn.delete_app(app_id, force=True)
                time.sleep(10)

        # allow a 404 at any point
        except marathon_api.exceptions.MarathonHttpError as e:
            if e.status_code != 404:
                raise

    def run(self, name, image, entrypoint, command):  # noqa
        """Run a one-off command"""
        self.fleet._create_container(name, image, command,
                                     copy.deepcopy(RUN_TEMPLATE),
                                     entrypoint=entrypoint)

        # wait for the container to get scheduled
        for _ in range(30):
            states = self.fleet._get_state(name)
            if states and len(states.get('states', [])) == 1:
                state = states.get('states')[0]
                break
            time.sleep(1)
        else:
            raise RuntimeError('container did not report state')
        machineID = state.get('machineID')

        # find the machine
        machines = self.fleet._get_machines()
        if not machines:
            raise RuntimeError('no available hosts to run command')

        # find the machine's primaryIP
        primaryIP = None
        for m in machines.get('machines', []):
            if m['id'] == machineID:
                primaryIP = m['primaryIP']
        if not primaryIP:
            raise RuntimeError('could not find host')

        # prepare ssh key
        file_obj = cStringIO.StringIO(base64.b64decode(self.fleet.pkey))
        pkey = paramiko.RSAKey(file_obj=file_obj)

        # grab output via docker logs over SSH
        ssh = paramiko.SSHClient()
        ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        ssh.connect(primaryIP, username="core", pkey=pkey)
        # share a transport
        tran = ssh.get_transport()

        def _do_ssh(cmd):
            chan = tran.open_session()
            # get a pty so stdout/stderr look right
            chan.get_pty()
            out = chan.makefile()
            chan.exec_command(cmd)
            rc, output = chan.recv_exit_status(), out.read()
            return rc, output

        # wait for container to start
        for _ in range(1200):
            rc, _ = _do_ssh('docker inspect {name}'.format(**locals()))
            if rc == 0:
                break
            time.sleep(1)
        else:
            raise RuntimeError('container failed to start on host')

        # wait for container to complete
        for _ in range(1200):
            _rc, _output = _do_ssh('docker inspect {name}'.format(**locals()))
            if _rc != 0:
                raise RuntimeError('failed to inspect container')
            _container = json.loads(_output)
            finished_at = _container[0]["State"]["FinishedAt"]
            if not finished_at.startswith('0001'):
                break
            time.sleep(1)
        else:
            raise RuntimeError('container timed out')

        # gather container output
        _rc, output = _do_ssh('docker logs {name}'.format(**locals()))
        if _rc != 0:
            raise RuntimeError('could not attach to container')

        # determine container exit code
        _rc, _output = _do_ssh('docker inspect {name}'.format(**locals()))
        if _rc != 0:
            raise RuntimeError('could not determine exit code')
        container = json.loads(_output)
        rc = container[0]["State"]["ExitCode"]

        # cleanup
        self.fleet._destroy_container(name)
        self.fleet._wait_for_destroy(name)

        # return rc and output
        return rc, output

    def attach(self, name):
        """
        Attach to a job's stdin, stdout and stderr
        """
        raise NotImplementedError

SchedulerClient = MarathonHTTPClient
