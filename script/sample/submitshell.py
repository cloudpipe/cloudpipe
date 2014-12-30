#!/usr/bin/env python

from __future__ import print_function

import multyvac
import sys

multyvac.config.set_key(api_key='admin', api_secret_key='12345', api_url='http://docker:8000/api')

jobs = {
    "stdout result": {
        "cmd": 'echo "success"',
    },
    "file result": {
        "cmd": 'echo "success" > /tmp/out',
        "_result_source": "file:/tmp/out",
    },
    "stdin": {
        "cmd": 'cat',
        "_stdin": "success",
    },
}

longest = 0
for name in jobs.keys():
    if len(name) > longest:
        longest = len(name)

success = 0
failure = 0

for (name, kwargs) in jobs.items():
    jid = multyvac.shell_submit(**kwargs)
    print("{:<{}}: job {} ...".format(name, longest, jid), end='')
    result = multyvac.get(jid).get_result().strip('\n')
    print(" result [{}]".format(result))
    if result == "success":
        success += 1
    else:
        failure += 1

print("{} pass / {} fail".format(success, failure))
if failure > 0:
    sys.exit(1)
