RNBin
=====

AWS S3 proxy library for put/get file.

## Concept

RNBin is proxy for save many files to AWS S3.
S3 request limit is per bucket(*). so RNBin uses multi buckets, and increase scale limit.
however, it can not scale dynamicaly. must do sizing first.

(*) [Request Rate and Performance Considerations](http://docs.aws.amazon.com/AmazonS3/latest/dev/request-rate-perf-considerations.html)

### RNBin do

- Put file to S3 with disbributed key name for S3 Request balancing

### RNBin do not

- Crypto:
Please encrypt file before request RNBin if you need encryption per file.
You can encrypt bucket with S3 settings. this is strage level encryption.

- Listing and Search:
RNBin do access to file using only the key(path)
Please create metadata index in front of RNBin application.

## functions

- Save file and Metadata, then return key
- Get file with key
- Get Metadata with key

## use case

- library in your go application (import in your golang program)
- microservice or other program language (http server)

## run http server

create buckets and then

```
./rnbin -region ap-northeast-1 -buckets=bucket1,bucket2,bucket3
```

running port 8000

## TODO

- logging
- http response format
- bucket checking
- testing
