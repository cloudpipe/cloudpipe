Cloudpipe
==========

Compute on demand in Docker containers.

Watch the [Cloudpipe Prototype Demo](https://www.youtube.com/watch?v=AGeALpR6DVc)

[![Build Status](https://travis-ci.org/cloudpipe/cloudpipe.svg?branch=master)](https://travis-ci.org/cloudpipe/cloudpipe)

## Hacking on the Cloudpipe backend server

 1. Install [Docker](https://docs.docker.com/installation/mac/) on your platform.
 2. Install [compose](https://docs.docker.com/compose/install/).
 3. Generate development TLS credentials by running `script/genkeys`.
 4. Run `docker-compose build && docker-compose up -d` to build and launch everything locally.

To run the tests, use `script/test`. You can also use `script/mongo` to connect to your local MongoDB
database.

### Running code against the system

For this iteration, we've implemented (some of) [multyvac's API](http://docs.multyvac.com/) allowing you to use `multyvac` for Python 2. We've created a fork that adapts to our base image and fixes some bugs evident when using the IPython/Jupyter Notebook.

```
pip install vac
```

:warning: If you already had `multyvac` installed, you'll likely want to delete `~/.multyvac`. Note that installing `vac` does overwrite the `multyvac` package.

Configure the client to connect to yours (default settings from compose shown here):

```
>>> import multyvac
>>> api_url = 'http://{}/v1'.format(<your_ip_endpoint>)
>>> multyvac.config.set_key(api_key='admin', api_secret_key='12345', api_url=api_url)
```

Create a Job

```
>>> def add(x,y):
...   sum = x + y
...   print("{x} + {y} = {sum}".format(x=x, y=y, sum=sum))
...   return sum
...
>>> job_id = multyvac.submit(add, 2, 3)
```

Retrieve the results

```
>>> job = multyvac.get(job_id)
>>> job.get_result()
5
>>> print(job.stdout)
2 + 3 = 5
```
