# golang-multithreaded-download
golang 实现多线程下载

* 通过一次 GET 请求获取到文件总体积
* 根据文件的体积和线程数 "平分" 每个线程需要下载的任务区间
* 下载任务通过设置 Range 请求头来请求想要的数据区间
