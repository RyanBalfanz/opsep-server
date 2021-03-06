# Op-Sep: Separate Your Data from your Keys
Use asymmetric cryptography to enforce rate-limits on your most sensitive data.
Operationally separate your data from your keys.
Reduce the damage of data-breaches 100x, and maintain an audit log to track exposure.

## Setup

### Fetch Repo
```bash
$ git clone git@github.com:opsep/opsep-server.git && cd opsep-server
```

### Create an RSA Keypair

```bash
$ openssl genrsa -out pem.priv 4096 && openssl rsa -in pem.priv -pubout -out crt.pub
Generating RSA private key, 4096 bit long modulus (2 primes)
...........................................++++
...............................++++
e is 65537 (0x010001)
writing RSA key
```
(this is how the keypairs in `insecure_certs` were generated)

### Run the Server
```bash
$ RSA_PRIVATE_KEY="$(cat insecure_certs/pem.priv)" go run *.go
```
(substitute your own private key)

### Confirm Server Online
This will also output the public configuration (your RSA private key is never extracted):
```bash
$ curl localhost:8080 
{
  "sqliteFilePath": "opsep.sqlite3",
  "serverHost": "localhost",
  "serverPort": "8080",
  "rsaPubKey": "-----BEGIN RSA PUBLIC KEY-----\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA7q4R3soRD2CrjL13OK6Y\nSBG8wpjP5sbfkL0QhpJMH87grlR2SS3CUnbYCOONzQiJ3OuKAViy/lMw1KsmG9Nn\nhAot2acg1iNyZRY33LR2jwmfFF+2iRp0itPQeOHY6GS8m3WLCMtC/kWUq0Bl5g1P\nYa9JXwSkTTRJunNH0TPk8uqwFeVhpT336M1H6ed105L8a8W3mpSwlwePron7pLf7\nwD32m9RT0nNdnHBDQCsUKS/Gdp+saLYWTgj0rpnQCe8f1p3g36Gm0gTzr3X0Adow\n8gIPfxO4HU/0cdL+Pw4mpcsWJ4531taRLLGb+a2la2zAUteYcS+8d4Nb8Omkbz39\nPylvKP6R1kHElqlF3BnwUp0AdcAvOLdeX8kYUlbKE8xwjHm/KwwleKlcAZDam7hC\nRw72JUQiod0E7My+SiZ3Ij5zKnxZXmAF5BX8T+YSqSzR4Qdp2QU9L9GgAZo/HPBN\nwME9v8usjEzrEItSSg3Nn10+J+ygsCqjrCT8CnSvD8wEyDSdO/Jly9DnWJ6B2HJE\nOc4wxWGFTCE0wiQOwC3IPNxFhuWun6/4tsEQcDs5XHaBXIHry5WCiVkjwa2pc95x\niXcfoQWr1A/jLe/MrZyN4yrgDK9mmQxxNzVfLj8S9NPjJMv+K7BKvtOmvoqsf13K\n6hYJGkAdR0d99DNFlllRm7cCAwEAAQ==\n-----END RSA PUBLIC KEY-----\n",
  "decryptsAllowedPerPeriod": 100,
  "periodInSeconds": 600
}
```

### Run Tests
See client info below.

---

## Use

The proper way to use opsep is with a client library that abstracts away all these complexities.
The information below is only for advanced users, for example those wishing to write their own client library (perhaps in a new langauge).
You can check out the python opsep client [here](https://github.com/opsep/opsep-server)

### Encryption

Encrypt a key locally (this would be randomly generated by your client library and not `0000...0000`):
```bash
$ TO_DECRYPT=$(echo "{\"key\":\"00000000000000000000000000000000\"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)
```

You can inspect the value stored in `TO_DECRYPT` like this:
```bash
$ echo $TO_DECRYPT
```

### Decryption

Make an API call to decrypt the file you just made (this is how your client library would later retrieve the randomly generated data-encryption-key it previously generated):
```bash
$ curl -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo $TO_DECRYPT)'"}' | python -m json.tool
{
    "keyRecovered": "00000000000000000000000000000000",
    "requestSHA256": "1e16673d9b0bb33e7cfb84d6c0bf96970f3cfc34c4f7f41987bd624c0912f69a",
    "ratelimitTotal": 100,
    "ratelimitRemaining": 99,
    "ratelimitResetsIn": 599
}
```

### Audit Logging for Decryption Requests
Note that when inspecting audit logs (individual records or bulk dumps), opsep doesn't store/dump sensitive data.


#### Query by Key Retrieval Ciphertext
Calculate the hash of your decryption instructions:
```bash
$ echo $TO_DECRYPT | base64 --decode | shasum -a 256
55a80d54fd68dea27f4186a9f6466082f02af25939546d974eb19c3eee4e4114  -
```
(your result will be different, as asymmetric encryption uses a randomly generated nonce each time it is run)

Query to see all past decryptions using those decryption instructions:
```bash
$ curl localhost:8080/api/v1/logs/55a80d54fd68dea27f4186a9f6466082f02af25939546d974eb19c3eee4e4114 | python -m json.tool
[
    {
        "ServerLogID": 18,
        "CreatedAt": "2020-08-11T18:10:09Z",
        "RequestSha256Digest": "55a80d54fd68dea27f4186a9f6466082f02af25939546d974eb19c3eee4e4114",
        "RequestIPAddress": "127.0.0.1",
        "RequestUserAgent": "curl/7.64.1",
        "ClientRecordID": null,
        "DeprecateAt": null,
        "RiskMultiplier": null
    }
]
```
There could be multiple instances of decrypting that data, as your client might request decryption multiple times.

#### Query by Client Record ID
Note that this is only possible if the `client_record_id` was originally encrypted as part of the `key_retrieval_ciphertext`.
```bash
$ sqlite3 opsep.sqlite3 -header -csv 'SELECT * FROM api_calls WHERE client_record_id = "aaaaaaaa-0000-bbbb-1111-cccccccccccc" LIMIT 2;'
id,created_at,request_sha256digest,request_ip_address,request_user_agent,response_dsha256digest,deprecate_at,client_record_id,risk_multiplier
3,"2020-08-11 17:42:54",c188a894fc1bc77ebd5872ec0f49f4d2f5876ea3aa7d6176258ea7d2fc1f0328,127.0.0.1,curl/7.64.1,9dd45edc9bf8afc0f06bd369da7e586169aaa2b0d616a3cdb9974344f7a5cab6,,aaaaaaaa-0000-bbbb-1111-cccccccccccc,
4,"2020-08-11 17:43:28",78d3ff5f0d6745f904959ef84a301024f6090e44309e6ab0ee195346e83d922e,127.0.0.1,curl/7.64.1,9dd45edc9bf8afc0f06bd369da7e586169aaa2b0d616a3cdb9974344f7a5cab6,,aaaaaaaa-0000-bbbb-1111-cccccccccccc,
```

#### Dump All Decryption Requests

Worried about a breach? See all decrypts as CSV:
```bash
$ sqlite3 opsep.sqlite3 -header -csv 'SELECT * FROM api_calls;'
```

---

## Advanced Features

### Key Deprecation
Pick an expiration date for your key when you generate it:
```bash
$ TO_DECRYPT=$(echo "{\"key\":\"00000000000000000000000000000000\", \"deprecate_at\":\"2020-01-01T12:00:00Z\"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)
$ curl -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo $TO_DECRYPT)'"}' | python -m json.tool
{
    "error_name": "DeprecatedDecryptionKeyError",
    "error_description": "Key to decrypt this payload marked as deprecated."
}
```
Note that your data can still be recovered if you have your Key Encryption Key, but it is not defualt accessible via this service.

### Risk Multiplier
Have specific records that you know are extra-sensitive?
You can make these count as multiple records for the purposes of your rate-limit:
```
$ TO_DECRYPT=$(echo "{\"key\":\"00000000000000000000000000000000\", \"risk_multiplier\":10}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)
$ curl -s -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo $TO_DECRYPT)'"}' | python -m json.tool
{
    "keyRecovered": "00000000000000000000000000000000",
    "requestSHA256": "33dac6df324b177ce26720eed545f97e69c9bce5e0caa083e62a665996509cec",
    "ratelimitTotal": 100,
    "ratelimitRemaining": 90,
    "ratelimitResetsIn": 599
}
```
`ratelimitRemaining` correctly fell from `100` to `90` in one API decryption requests.


### Client Record ID
Tracking decryption using `key_retrieval_ciphertext` can be cumbersome.
Even easier, at the time of encryption you can save your local client record ID. Note that regardless of underlying type this must be saved as a string (normal for UUIDs, but counterintuitive for INTs).
```bash
$ TO_DECRYPT=$(echo "{\"key\":\"00000000000000000000000000000000\", \"client_record_id\":\"aaaaaaaa-0000-bbbb-1111-cccccccccccc\"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)
$ curl -s -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo $TO_DECRYPT)'"}' | python -m json.tool
{
    "keyRecovered": "00000000000000000000000000000000",
    "requestSHA256": "6e7679934a8122b6f4e91d7a5da09e37dbc31777c714b23294eeb329a3a586cf",
    "ratelimitTotal": 100,
    "ratelimitRemaining": 99,
    "ratelimitResetsIn": 599
}
```

What's particularly powerful is that you can query your Opsep server by these IDs.
See Audit Logging below.

### Rate-Limit Test
If you want a rough test of 429-ing, you can do this:
```bash
$ for i in {1..99}; do curl [...] "http://localhost:8080/api/v1/decrypt" ; done
```
### Load Testing
Run server with a high threshold of decryptions:
```bash
RSA_PRIVATE_KEY="$(cat insecure_certs/pem.priv)" DECRYPTS_PER_PERIOD=99999 go run *.go
```

Before and after your load test, query your `sqlite` DB to confirm the correct # of records were inserted:
```bash
$ sqlite3 opsep.sqlite3 'SELECT COUNT(1) from api_calls;'
32743
```

#### Using Curl
Make requests as client:
```bash
$ TO_DECRYPT=$(echo "{\"key\":\"00000000000000000000000000000000\"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)
$ time for i in {1..100}; do curl -s -o /dev/null -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo $TO_DECRYPT)'"}' ; done
real	0m1.969s
user	0m0.325s
sys	0m0.495s
```
(On Windows replace `/dev/null` with `nul`)

#### Using Apache Bench
Download [ab](https://httpd.apache.org/docs/2.4/programs/ab.html).


Note that [you must use `127.0.0.1` instead of `localhost`](https://www.bram.us/2020/02/20/fixing-ab-apachebench-error-apr_socket_connect-invalid-argument-22/).
```bash
$ TO_DECRYPT=$(echo "{\"key\":\"00000000000000000000000000000000\"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)
$ echo "{\"key_retrieval_ciphertext\":\"$( echo $TO_DECRYPT )\"}" > ab.json
$ ab -p ab.json -T "application/json" -c 2 -n 100 http://127.0.0.1:8080/api/v1/decrypt 
This is ApacheBench, Version 2.3 <$Revision: 1843412 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient).....done


Server Software:        
Server Hostname:        127.0.0.1
Server Port:            8080

Document Path:          /api/v1/decrypt
Document Length:        213 bytes

Concurrency Level:      2
Time taken for tests:   0.563 seconds
Complete requests:      100
Failed requests:        0
Total transferred:      33700 bytes
Total body sent:        86600
HTML transferred:       21300 bytes
Requests per second:    177.77 [#/sec] (mean)
Time per request:       11.250 [ms] (mean)
Time per request:       5.625 [ms] (mean, across all concurrent requests)
Transfer rate:          58.50 [Kbytes/sec] received
                        150.34 kb/s sent
                        208.85 kb/s total

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       1
Processing:     9   11   1.7     10      15
Waiting:        9   11   1.7     10      15
Total:          9   11   1.7     11      15

Percentage of the requests served within a certain time (ms)
  50%     11
  66%     12
  75%     12
  80%     13
  90%     14
  95%     14
  98%     15
  99%     15
 100%     15 (longest request)
```

### One-Liner
Regular:
```bash
$ curl -s -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo "{\"key\":\"00000000000000000000000000000000\"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)'"}' | python -m json.tool
{
    "keyRecovered": "00000000000000000000000000000000",
    "requestSHA256": "df9e875c1670896d1213f6fa401f12ce4bcfdc047d0ab050620f70626db53b89",
    "ratelimitTotal": 100,
    "ratelimitRemaining": 99,
    "ratelimitResetsIn": 599
}
```

Expired key:
```bash
$ curl -s -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo "{\"key\":\"00000000000000000000000000000000\", \"deprecate_at\":\"2020-01-01T12:00:00Z\"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)'"}' | python -m json.tool
{
    "error_name": "DeprecatedDecryptionKeyError",
    "error_description": "Key to decrypt this payload marked as deprecated."
}
```

Fancy. In addition to `key` we include `deprecate_at`, `client_record_id`, and `risk_multiplier`):
```bash
$ curl -s -X POST localhost:8080/api/v1/decrypt -H 'Content-Type: application/json' -d '{"key_retrieval_ciphertext":"'$(echo "{\"key\":\"00000000000000000000000000000000\", \"deprecate_at\":\"2030-01-01T12:00:00Z\", \"client_record_id\":\"aaaaaaaa-0000-bbbb-1111-cccccccccccc\", \"risk_multiplier\":"3"}" | openssl pkeyutl -encrypt -pubin -inkey insecure_certs/crt.pub -pkeyopt rsa_padding_mode:oaep -pkeyopt rsa_oaep_md:sha256 -pkeyopt rsa_mgf1_md:sha256 | base64)'"}' | python -m json.tool
{
    "keyRecovered": "00000000000000000000000000000000",
    "requestSHA256": "e505f29ab3da7fa6e02ef7cc2ff42bcef66cb4e2fff814ae8d36344306a07a40",
    "ratelimitTotal": 100,
    "ratelimitRemaining": 99,
    "ratelimitResetsIn": 599
}
```

### Extract Public Key from Opsep Server
To be sure that you're encrypting your data locally with the correct pubkey:
```bash
$ curl localhost:8080 | python3 -c "import sys, json; print(json.load(sys.stdin)['rsaPubKey'].strip())" | awk '{gsub(/\\n/,"\n")}1' > crt.pub
```

### Advanced Deployment Options 
Other environmental variable options include `SQLITE_FILEPATH`, `SERVER_HOST`, `SERVER_PORT`, `DECRYPTS_PER_PERIOD`, and `PERIOD_IN_SECONDS`.
See [config.go](config.go) for more info.


## HSM
RSA is compatible with all major HSMs.
You're on your own for implementating that.

## Help
File an issue or contact me at opsep@michaelflaxman.com if you're stuck.
