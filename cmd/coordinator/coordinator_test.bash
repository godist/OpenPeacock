printf "Building ... "
if go install \
    github.com/wangkuiyi/prism/prism \
    github.com/wangkuiyi/phoenix/cmd/coordinator \
    github.com/wangkuiyi/phoenix/cmd/sampler \
    github.com/wangkuiyi/phoenix/cmd/loader; then
    echo "Done."
else
    echo "Failed."
    exit
fi

printf "Start Prism ... "
killall prism
$GOPATH/bin/prism &
echo "Done."

printf "Mimic deployment ... "
rm -rf /tmp/ready_to_fly
mkdir -p /tmp/ready_to_fly/deploy
cp $GOPATH/bin/coordinator /tmp/ready_to_fly/deploy/
cp $GOPATH/bin/sampler     /tmp/ready_to_fly/deploy/
cp $GOPATH/bin/loader      /tmp/ready_to_fly/deploy/

echo "Start Phoenix coordinator ... "
killall coordinator
killall sampler
killall loader
$GOPATH/bin/coordinator -config='{
    "JobName":"ready_to_fly",
    "DeployDir":"file:/tmp/ready_to_fly/deploy",
    "LogDir":"file:/tmp/ready_to_fly/log",
    "CorpusDir":"file:/tmp/ready_to_fly/corpus/",
    "VocabFile":"file:/tmp/ready_to_fly/vocab",
    "Master":"",
    "Retry":5,
    "Squads":[
        {"Name":"squad0",
         "Coordinator":"localhost:10010",
         "Loaders":["localhost:10020","localhost:10021"],
         "Samplers":["localhost:10030","localhost:10031"]
        }
    ],
    "Aggregators":[],
    "Machines":null,
    "NumVShards":2,
    "JobDir":"file:/tmp/ready_to_fly/",
    "NumTopics":2,
    "TopicPrior":0.1,
    "WordPrior":0.01
}' -addr="localhost:10010"
