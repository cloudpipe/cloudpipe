#!/usr/bin/env python

from __future__ import print_function

import multyvac

multyvac.config.set_key(api_key='admin', api_secret_key='12345', api_url='http://docker:8000/api')

def add(a, b):
    return a + b

jid = multyvac.submit(add, 3, 4)
result = multyvac.get(jid).get_result()
print("result = {}".format(result))
