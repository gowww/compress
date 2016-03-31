# [![gowww](https://avatars.githubusercontent.com/u/18078923?s=20)](https://github.com/gowww) compress

[![Build Status](https://travis-ci.org/gowww/compress.svg?branch=master)](https://travis-ci.org/gowww/compress)
[![Coverage Status](https://coveralls.io/repos/github/gowww/compress/badge.svg?branch=master)](https://coveralls.io/github/gowww/compress?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/gowww/compress)](https://goreportcard.com/report/github.com/gowww/compress)
[![GoDoc](https://godoc.org/github.com/gowww/compress?status.svg)](https://godoc.org/github.com/gowww/compress)

Package [compress](https://godoc.org/github.com/gowww/compress) provides a clever gzip compressing handler.

It takes care to not handle small contents, or contents that are already compressed (like JPEG, MPEG or PDF).  
Trying to gzip them not only wastes CPU but can potentially increase the response size.

Make sure to include this handler above any other handler that alter the response body.
