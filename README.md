# Mini-Scan

Hello!

As you've heard by now, Censys scans the internet at an incredible scale. Processing the results necessitates scaling horizontally across thousands of machines. One key aspect of our architecture is the use of distributed queues to pass data between machines.

---

The `docker-compose.yml` file sets up a toy example of a scanner. It spins up a Google Pub/Sub emulator, creates a topic and subscription, and publishes scan results to the topic. It can be run via `docker compose up`.

Your job is to build the data processing side. It should:

1. Pull scan results from the subscription `scan-sub`.
2. Maintain an up-to-date record of each unique `(ip, port, service)`. This should contain when the service was last scanned and a string containing the service's response.

> **_NOTE_**
> The scanner can publish data in two formats, shown below. In both of the following examples, the service response should be stored as: `"hello world"`.
>
> ```javascript
> {
>   // ...
>   "data_version": 1,
>   "data": {
>     "response_bytes_utf8": "aGVsbG8gd29ybGQ="
>   }
> }
>
> {
>   // ...
>   "data_version": 2,
>   "data": {
>     "response_str": "hello world"
>   }
> }
> ```

Your processing application should be able to be scaled horizontally, but this isn't something you need to actually do. The processing application should use `at-least-once` semantics where ever applicable.

You may write this in any languages you choose, but Go would be preferred.

You may use any data store of your choosing, with `sqlite` being one example. Like our own code, we expect the code structure to make it easy to switch data stores.

Please note that Google Pub/Sub is best effort ordering and we want to keep the latest scan. While the example scanner does not publish scans at a rate where this would be an issue, we expect the application to be able to handle extreme out of orderness. Consider what would happen if the application received a scan that is 24 hours old.

cmd/scanner/main.go should not be modified

---

Please upload the code to a publicly accessible GitHub, GitLab or other public code repository account. This README file should be updated, briefly documenting your solution. Like our own code, we expect testing instructions: whether it’s an automated test framework, or simple manual steps.

To help set expectations, we believe you should aim to take no more than 4 hours on this task.

We understand that you have other responsibilities, so if you think you’ll need more than 5 business days, just let us know when you expect to send a reply.

Please don’t hesitate to ask any follow-up questions for clarification.

## Arvind's Mini-Scan
I have chosen BigTable as the data store for this solution. While I have no experience working with Google Cloud, it seems to be that BigTable is the only solution out there from Google Cloud to handle high volume concurrent writes. This is different in the AWS world, where the NoSQL document db (DynamoDB) can handle this sort of traffic. Firestore did not seem to be designed for such use cases. I also liked the fact that the gcloud emulator is so easy to use, even for a noob like myself.

Unfortunatly, I was unable to get the BigTable docker container working. It consistenly failed the healthcheck. My attempt at setting all this up is in docker-compose.yml

The instructions below are for running the applications locally using the gcloud emulator. Installation instructions are available at https://docs.cloud.google.com/sdk/docs/install. I also verified that the data was being written to BigTable using the cbt tool - installation instructions at https://docs.cloud.google.com/bigtable/docs/cbt-overview#installing

This effort took me about 10 hours. I also used Gemini AI on my web browser to research many pointed questions I had about google cloud and the usage of the golang sdk for bigtable and pubsub. While the code is my own and not AI copy / pasta, some of the information even it may have been showing outdated stuff, helped me lot to figure things out.

## Steps
0. Back up any existing .cbtrc file in your home directory and replace it with one that looks like this (there is a possibility that replacing is unnecessary - pardon my inexperience with cbt and google cloud)
```
project = test-project
instance = test-instance
```

1. Build the applications
```bash
$ go build -o ./build/scanner ./cmd/scanner/...

$ go build -o ./build/processor ./cmd/processor/...

$ go build -o ./build/admin ./cmd/admin/...
```

2. Run the bigtable emulator in a separate terminal
```bash
$ gcloud beta emulators bigtable start --host-port=0.0.0.0:8086
```

3. Run the pubsub emulator in a separate terminal
```bash
$ gcloud beta emulators pubsub start --project test-project --host-port 0.0.0.0:8085
```

4. Create the topic and the subscription. Run the admin tool followed by the processor in a separate terminal
```bash
$ curl -X PUT http://127.0.0.1:8085/v1/projects/test-project/topics/scan-topic

$ curl -X PUT http://127.0.0.1:8085/v1/projects/test-project/subscriptions/scan-sub \
     -H "Content-Type: application/json" \
     -d '{ "topic": "projects/test-project/topics/scan-topic" }'
    
$ BIGTABLE_EMULATOR_HOST=127.0.0.1:8086 ./build/admin

$ BIGTABLE_EMULATOR_HOST=127.0.0.1:8086 PUBSUB_EMULATOR_HOST=127.0.0.1:8085 ./build/processor
```

5. Run the scanner in a separate terminal
```bash
$ PUBSUB_EMULATOR_HOST=127.0.0.1:8085 ./build/scanner
```

6. Use the cbt tool to examing the contents of the ScanData table in a separate terminal
```bash
$ export BIGTABLE_EMULATOR_HOST=127.0.0.1:8086 # one time step

$ cbt ls

$ cbt read ScanData
```
