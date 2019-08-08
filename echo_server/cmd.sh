
# 生成指定长度字符串
function _gen_length_string(){
    s=""
    for ((i=0;i<$1;i++));
    do
        ran=$RANDOM
        # ran=1`date +%1N`
        d=$((${ran}%16))
        s=${s}`printf %x $d`
    done
    echo $s
}

function consistgenstring(){
    cnt=$1
    while :
    do 
        str=`_gen_length_string $cnt`
        echo $str
        echo $str >> $tmpfile
        cnt=$[ cnt + 1 ]
        sleep $2
    done    
}

function _printStop(){
    sleep 1
    stoplen=`tail -1 $tmpfile |wc -L`
    echo stop len: $stoplen, total `wc -c $tmpfile`
}

function xnc(){
    cnt=$1
    interval=$2
    shift 2

    tmpfile=`mktemp`
    # tmpfile=/tmp/1.txt
    # echo $tmpfile
    echo start len: $cnt
    trap "_printStop; exit" SIGINT

    # 在pipe之后的处理不会改变原来的值，因为新建一个进程
    consistgenstring $cnt $interval | nc  $@
}

# 执行命令行
$@