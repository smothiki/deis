import cStringIO
import base64
import copy
import json
import httplib
import re
import time
import string
from django.conf import settings

POD_TEMPLATE = '''{
  "id": "$id",
  "kind": "Pod",
  "apiVersion": "$version",
  "desiredState": {
    "manifest": {
      "version": "$version",
      "id": "$id",
      "containers": [{
        "name": "$id",
        "image": "$image",
        "ports": [{
          "containerPort": 80,
          "hostPort": 80
        }]
      }]
    }
  },
  "labels": {
    "name": "$id",
    "environment": "testing"
  }
}'''

RETRIES = 3

class KubeHTTPClient():

    def __init__(self, target, auth, options, pkey):
        self.target = settings.K8S_MASTER
        self.port = "8080"
        self.registry = settings.REGISTRY_HOST+":"+settings.REGISTRY_PORT
        self.apiversion = "v1beta1"
        self.conn = httplib.HTTPConnection(self.target+":"+self.port)

    # container api

    def create(self, name, image, command, **kwargs):
        l = {}
        l["id"]=name
        l["version"]=self.apiversion
        l["image"]=self.registry+"/"+image
        template=string.Template(POD_TEMPLATE).substitute(l)
        headers = {'Content-Type': 'application/json'}
        #http://172.17.8.100:8080/api/v1beta1/pods
        print copy.deepcopy(template)
        for attempt in range(RETRIES):
            try:
                self.conn.request('POST', '/api/'+self.apiversion+'/'+'pods',
                          headers=headers, body=copy.deepcopy(template))
                resp = self.conn.getresponse()
                data = resp.read()
                if not 200 <= resp.status <= 299:
                    errmsg = "Failed to retrieve unit: {} {} - {}".format(
                        resp.status, resp.reason, data)
                    raise RuntimeError(errmsg)
                return data
            except:
                if attempt >= (RETRIES - 1):
                    raise

    def start(self, name):
        """
        Start a container
        """
        return

    def stop(self, name):
        """
        Stop a container
        """
        return

    def destroy(self, name):
        """
        Destroy a container
        """
        return

    def run(self, name, image, entrypoint, command):
        """
        Run a one-off command
        """
        # dump input into a json object for testing purposes
        return 0, json.dumps({'name': name,
                              'image': image,
                              'entrypoint': entrypoint,
                              'command': command})

    def attach(self, name):
        """
        Attach to a job's stdin, stdout and stderr
        """
        return StringIO(), StringIO(), StringIO()

SchedulerClient = KubeHTTPClient
