#!/usr/bin/env python

import multyvac

multyvac.config.set_key(api_key='admin', api_secret_key='12345', api_url='http://docker:8000/api')

jid = multyvac.shell_submit(cmd='for i in {1..10}; do echo $i && sleep 10; done')
print("Submitted job [{}].".format(jid))

job = multyvac.get(jid)
result = job.get_result()
print("Result: [{}]".format(result))
