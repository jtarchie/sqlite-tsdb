# sqlite times series databases with s3 cold storage

Not a clever name for now, calling it by it's intention.

## Letter from the Editor

Querying is a crucial aspect of working with time-series databases, and various
query languages have been developed to support this task, such as
[Prometheus Query Language (promQL)](https://prometheus.io/docs/prometheus/latest/querying/basics/),
[Search Processing Language (SPL)](https://docs.splunk.com/Documentation/SplunkCloud/9.0.2209/SearchTutorial/Usethesearchlanguage),
[CloudWatch Logs Query Syntax (CWL)](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html)).
These languages provide users with flexibility in exploring their data and can
be extended to meet specific needs.

However, as a seasoned engineer, I have often found myself translating SQL
queries into these other languages, which may not have a one-to-one mapping but
can help me understand the shape of the data and the queries I want to write.

This documentation, which is driven by "README" development, is motivated by my
desire to be able to query time-series data more effectively and will be updated
as changes and discussions take place.

## Problem Statement
