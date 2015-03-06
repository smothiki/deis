import cStringIO
import base64
import copy
import json
import httplib
import re
import time
import string
import urllib

POD_TEMPLATE = '''{
  "id": "$id",
  "kind": "Pod",
  "apiVersion": "$version",
  "desiredState": {
    "manifest": {
      "version": "$version",
      "id": "hello",
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
        l["id"]="hello"
        l["version"]=version
        l["image"]="golang"
        template=string.Template(POD_TEMPLATE).substitute(l)
        headers = {'Content-Type': 'application/json'}
            #http://172.17.8.100:8080/api/v1beta1/pods
        print copy.deepcopy(template)
        self.conn.request('POST', '/api/'+self.apiversion+'/'+'pods',
                              headers=headers, body=copy.deepcopy(template))
        resp = self.conn.getresponse()
        data = resp.read()
        print data



j=KubeHTTPClient("172.17.8.101","8080","v1beta1")

j._create_pod("redis","Pod","v1beta1","redis-jaffa")
