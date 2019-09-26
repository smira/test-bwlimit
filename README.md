bandwidth limiter
=================

`bandwidth.LimitedReader` can be used to wrap any `io.Reader` to provide bandwidth limit.
Multiple `bandwidth.Limit` instances might be attached to it to implement global, per-connection limits
or any other kinds of bandwidth limiting.

Similar to `Reader`, `bandwidth.LimitedWriter` wraps `io.Writer` and provides bandwidth limits.

Implementation is based on `golang.org/x/time/rate/limit` package.

Unit-tests verify that bandwidth stays within 5% of the specified limit.
