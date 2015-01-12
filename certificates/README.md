# Nice Try

...but I don't want to even store real *development* TLS credentials in here. Run `script/genkeys` to populate it locally, or check out [the Docker article about configuring TLS support](https://docs.docker.com/articles/https/) if you want to do it yourself. :sparkles:

By default, `script/genkeys` will create a file in this directory called "dev.password" containing the (automatically generated, random) password used by all of the keys and certificates. If you want to use your own instead, repeat it on two lines of a text file with that name.
