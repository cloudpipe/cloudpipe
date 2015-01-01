Cloud Pipe
==========

Compute on demand in Docker containers.

[![Build Status](https://travis-ci.org/cloudpipe/cloudpipe.svg?branch=master)](https://travis-ci.org/cloudpipe/cloudpipe)

## Getting Started

 1. Install [Docker](https://docs.docker.com/installation/mac/) on your platform.
 2. Install [fig](http://www.fig.sh/install.html).
 3. Run `fig build && fig up -d` to build and launch everything locally.

To run the tests, use `script/test`. You can also use `script/mongo` to connect to your local MongoDB
database.

### Running code against the system

Install `multyvac` for Python 2:

```
pip install multyvac
```

Configure the client to connect to yours (default settings from fig shown here):

```
>>> import multyvac
>>> api_url = 'http://{}/v1'.format(<your_ip_endpoint>) 
>>> multyvac.config.set_key(api_key='admin',
...                         api_secret_key='12345',
...                         api_url=api_url)
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
