go install && \
$GOPATH/bin/multithread \
    -vocab=../singlethread/testdata/vocab \
    -corpus=../singlethread/testdata/corpus \
    -topics=2 \
    -readable=/tmp/r \
    2>/dev/null

E='Topic 00000 Nt 00004: orange (2) banana (1) apple (1)
Topic 00001 Nt 00004: cat (2) tiger (1) dog (1)'

R=$(cat /tmp/r)

if [[ "$R" != "$E" ]]; then
    echo "Expecting $E"
    echo "got $R"
    exit -1
fi

echo "Test passed"
