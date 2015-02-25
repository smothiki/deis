import cStringIO
import base64
import copy
import json
import httplib
import re
import time
import string


POD_TEMPLATE = '''{
  "id": "$id",
  "kind": "$pod",
  "apiVersion": "$version",
  "desiredState": {
    "manifest": {
    "version": "$version",
    "id": "$id",
    "containers": [{
      "name": "master",
      "image": "dockerfile/redis",
      "ports": [{
        "containerPort": 6371,
        "hostPort": 6371
      }]
    }]
    }
  },
  "labels": {
  "name": "$name"
  }
}'''


class KubeHTTPClient():

      def __init__(self, ipaddr,port,apiversion):
        self.ipaddr = ipaddr
        self.port = port
        self.apiversion = apiversion
        self.conn = httplib.HTTPConnection(self.ipaddr+":"+self.port)

        # single global connection
      def testurl(self):

        print "hello"
        self.conn.request('GET','/')
        resp = self.conn.getresponse()
        print resp.read()

      def _get_pods(self):
        self.conn.request('GET','/api/'+self.apiversion+'/'+'pods')
        resp = self.conn.getresponse()
        print resp.read()
        print "headers"
        print resp.getheaders()

      def _get_pod(self,podId):
        self.conn.request('GET','/api/'+self.apiversion+'/'+'pods/'+podId)
        resp = self.conn.getresponse()
        print resp.read()
        print "headers"
        print resp.getheaders()

      def _create_pod(self,id,pod,version,name):
        l = {}
        l["id"]=id
        l["pod"]=pod
        l["version"]=version
        l["name"]=name
        template=string.Template(POD_TEMPLATE).substitute(l)
        headers = {'Content-Type': 'application/json'}
        #http://172.17.8.100:8080/api/v1beta1/pods
        self.conn.request('POST', '/api/'+self.apiversion+'/'+'pods',
                          headers=headers, body=copy.deepcopy(template))
        resp = self.conn.getresponse()
        data = resp.read()
        print data



j=KubeHTTPClient("172.17.8.100","8080","v1beta1")
j._create_pod("redis","Pod","v1beta1","redis-jaffa")
