printf "Building ... "
if go install \
    github.com/wangkuiyi/phoenix/cmd/aggregator; then
    echo "Done."
else
    echo "Failed."
    exit
fi

echo "Start Phoenix aggregator ... "
killall aggregator
$GOPATH/bin/aggregator -config='{
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
    "Aggregators":["localhost:10040", "localhost:10041"],
    "Machines":null,
    "NumVShards":2,
    "JobDir":"file:/tmp/ready_to_fly/",
    "NumTopics":2,
    "TopicPrior":0.1,
    "WordPrior":0.01
}' -addr="localhost:10041"
