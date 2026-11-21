# Spec 000: Configuration reader for hubx-api (smoke test)
Source: user description

## What and why
hubx-api must read its configuration following constitution Principle III "Configuration", so that the
port and future settings are not hard-coded and secrets never live in files.

## Acceptance (Gherkin)
- Given no env var and no file, When config loads, Then `server.port` is the code default "8081".
- Given `application.properties` with `server.port=9000`, When config loads, Then port is "9000".
- Given `application.properties` with `server.port = 9300` (spaces around `=`), When config loads, Then port is "9300".
- Given APP_PROFILE=dev and `application-dev.properties` with `server.port=9100`, Then port is "9100".
- Given env `SERVER_PORT=9200` (or alias `PORT`), Then port is "9200" regardless of files.
- Given a malformed line (no `=`), When config loads, Then an error is returned.
- Given a missing file, When config loads, Then defaults are used (warning logged), no error.
