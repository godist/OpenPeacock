echo "Build Prism and Phoenix ..."
if go install github.com/wangkuiyi/prism/prism \
    github.com/wangkuiyi/phoenix/cmd/master \
    github.com/wangkuiyi/phoenix/cmd/aggregator \
    github.com/wangkuiyi/phoenix/cmd/coordinator  \
    github.com/wangkuiyi/phoenix/cmd/loader  \
    github.com/wangkuiyi/phoenix/cmd/sampler; then
    echo Build succeeded
else
    echo Build failed
    exit
fi

echo "Start Prism ..."
killall prism
$GOPATH/bin/prism &

echo "Prepare corpus and vocab ..."
rm -rf /tmp/ready_to_fly
mkdir -p /tmp/ready_to_fly
cp -r \
    $GOPATH/src/github.com/wangkuiyi/phoenix/cmd/master/testdata/corpus \
    /tmp/ready_to_fly/
cp \
    $GOPATH/src/github.com/wangkuiyi/phoenix/cmd/master/testdata/vocab \
    /tmp/ready_to_fly/

echo "Start Phoenix ..."
killall master
killall coordinator
killall sampler
killall loader
killall aggregator
$GOPATH/bin/master \
    -config_file=file:$GOPATH/src/github.com/wangkuiyi/phoenix/cmd/master/example.conf
