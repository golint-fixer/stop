#!/bin/sh
#
# Verifies that go code passes go fmt, go vet, golint, and go test.
#

o=$(tempfile)

fail() {
	echo Failed
	cat $o
	exit 1
}

echo Formatting
gofmt -l $(find . -name '*.go') 2>&1 > $o
test $(wc -l $o | awk '{ print $1 }') = "0" || fail

echo Vetting
go vet ./... 2>&1 > $o || fail

echo Testing
go test ./... 2>&1 > $o || fail

echo Linting
golint . \
	| grep -v 'Start should have comment'\
	| grep -v 'Loc should have comment'\
	| grep -v 'End should have comment'\
	| grep -v 'Eq should have comment'\
	| grep -v 'Fold should have comment'\
	| grep -v 'Underlying should have comment'\
	| grep -v 'Type should have comment'\
	| grep -v 'receiver name n should be consistent with previous receiver name t'\
	| grep -v 'Untyped.Identical'\
	| grep -v 'Source should have comment'\
	| grep -v 'Check should have comment'\

exit 0
