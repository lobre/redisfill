# redisfill

Tool to fill Redis with random data

```
$ ./redisfill -h
Usage of ./redisfill:
  -db int
        Redis database number
  -host string
        Redis host (default "localhost")
  -length int
        Length of generated values (default 50000)
  -max int
        Max memory in Mo (default 1000)
  -pass string
        Redis password
  -port string
        Redis port (default "6379")
  -prefix string
        Prefix to append to keys
  -workers int
        Number of parallel workers (default 100)
```
