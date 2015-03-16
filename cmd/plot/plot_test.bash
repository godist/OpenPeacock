plotfile=/tmp/$RANDOM.png
go install github.com/wangkuiyi/phoenix/cmd/plot && \
    $GOPATH/bin/plot -log=./testdata/nohup.out -plot=$plotfile && \
    if cmp ./testdata/nohup.png $plotfile; then
    echo "Passed"
    else
    echo "Generated file ($plotfile) does not match ./testdata/nohup.png"
    exit -1
    fi \
        || echo "Failed to build and run plot"
