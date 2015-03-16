#Phoenix Trainer使用方法

##训练数据的准备

训练数据包括两部分：

1. 训练语料(training corpus)：是一个文本文件，其中每一行是一个“训练文档”，包括用空格分开的多个字符串，其中每个字符串是一个token。

2. 词汇表(vocabulary)：是一个文本文件，其中每一行包括一个token。词汇表其实是一个白名单——corpus中的tokens，如果不在词汇表里，会被trainer忽略掉。通常，stopwords和typos应该不在词汇表里。

token不一定需要是单词，也可以是各种id。比如，如果一个训练文档对应一次用户购物行为，那么token可以对应这次购买的货品。

如果训练预料就是文本，那么需要先分词，然后统计词频，去除其中高频词（很可能是stopwords），去除其中低频词（只出现了一两次的词很可能是typos），去除根据先验知识应该是没有用的词（比如象声词——哦，啊，呢）。

Phoenix在训练语料充足的情况下，不依赖很强的分词器（能识别众多长词，比如“中华人民共和国”）。在我们的一个实验里，分词器不能识别“美雅士”这个词，于是给分成了三个单字：“美“”雅“”士”。但是Phoenix学出来一个latent topic，正好就包含这三个单字。这说明Phoenix本身就能学出来长词对应的概念。Phonix默认使用这个Go语言写的分词器：https://github.com/huichen/sego。

##训练过程

一个linux-amd64版本的训练程序（trainer）已经放在

    ubuntu@54.186.108.229:/home/ubuntu/wyi/usr/bin/multithread
    
基本用法如下：

    > ssh ubuntu@54.186.108.229
    > ~/wyi/usr/bin/multithread -corpus=/tmp/corpus.txt -vocab=/tmp/vocab.txt -topics=100
    
其中 `/tmp/corpus.txt`是上文中描述的训练语料文件；`/tmp/vocab.txt`是上文中说的词汇表文件；`100`是我们希望学得的latent topic的个数。在内存充裕的情况下，我们不妨把`-topics`设置得大一些，大于我们期待的latent topic个数。

##训练结果

目前，`multithread`训练程序输出的是一个人可读（但是机器不可用）的模型文件。这个文件主要是让用户肉眼观测每个latent topic是否靠谱。

我今晚会加上一个选项，直接输出标准模型，让inference（解释程序）可以用来解释任何输入对应的latent topics的。

##效果检验

Latent topic model的最重要的效果检验方法就是肉眼看学得的latent topics是否靠谱。

其他检验方法包括计算每个topics的PMI，然后略掉那些PMI比较小的topics。我们随后会开发计算PMI的工具。

上述两部检验之后，我们通常会进入应用检验——根据实际应用，做一些离线实验，以检测效果。

##参数调节

如果训练得到的模型不尽如人意，可以参考调节更多参数。参数列表请参见源码文件：`github.com/wangkuiyi/phoenix/cmd/multithread/multithread.go`。
