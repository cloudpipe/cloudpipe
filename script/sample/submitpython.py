#!/usr/bin/env python

# CLOUDPIPE_URL=http://`echo $DOCKER_HOST | cut -d ":" -f2 | tr -d "/"`:8000/v1 python2 script/sample/submitpython.py

from __future__ import print_function

import multyvac

import os
# Grab from the CLOUDPIPE_URL environment variable, otherwise assume they have
# /etc/hosts configured to point to their docker
api_url = os.environ.get('CLOUDPIPE_URL', 'http://docker:8000/v1')

multyvac.config.set_key(api_key='admin', api_secret_key='12345', api_url=api_url)

def add(a, b):
    return a + b

jid = multyvac.submit(add, 3, 4)
result = multyvac.get(jid).get_result()
print("added {} and {} to get {}... in the cloud!".format(3,4,result))
