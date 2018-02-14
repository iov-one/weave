# ORM

Some ideas for an easy to use db model.

Break state space into prefixed sections called Buckets.
* Each bucket contains only one type of object.
* It has a primary index (which may be composite),
and may possess secondary indexes.
* It may possess one or more secondary indexes (1:1 or 1:N)
* Easy queries for one and iteration.

For inspiration, look at [storm](https://github.com/asdine/storm) built on top of [bolt kvstore](https://github.com/boltdb/bolt#using-buckets).
* Do not use so much reflection magic. Better do stuff compile-time static, even if it is a bit of boilerplate.
* Consider general usability flow from that project
