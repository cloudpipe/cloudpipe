#!/usr/bin/env python

from __future__ import print_function

import multyvac
import time
import sys

multyvac.config.set_key(api_key='admin', api_secret_key='12345', api_url='http://docker:8000/v1')

def longtime(seconds):
    print("Getting started")
    sys.stdout.flush()
    for i in xrange(0, seconds):
        time.sleep(1)
        print("{} seconds".format(i))
        sys.stdout.flush()

jid = multyvac.submit(longtime, 30)
time.sleep(5)
multyvac.kill(jid)
time.sleep(1)
j = multyvac.get(jid)
print("job = {}, status = {}".format(j, j.status))
