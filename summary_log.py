#!/usr/bin/env python
# coding: utf-8
from __future__ import print_function

u"""アクセスログのサマリを取得するスクリプト.
\treqtime:実数\t
\tout:整数\t
\trequest:リクエストライン\t
を見つけて処理する. それ以外の部分は LTSV にしなくても別にいい.

nginxの場合

    log_format  isucon '$time_local $msec\t$status\treqtime:$request_time\t'
                       'in:$request_length\tout:$bytes_sent\trequest:$request\t'
                       'acceptencoding:$http_accept_encoding\treferer:$http_referer\t'
                       'ua:$http_user_agent';

Apache の場合


    LogFormat "%{%T}t\t%>s\treqtime:%D\tin:%I\tout:%O\trequest:%r\tacceptencoding:%{Accept-Encoding}i\treferer:%{Referer}i\tua:%{User-Agent}i" isucon
    CustomLog "logs/access_log" isucon
"""

import re
import sys
import fileinput
from collections import defaultdict


def summary_map(title, m, cnt=None, reverse=True):
    m = list(m.items())
    m.sort(key=lambda x: x[1], reverse=reverse)
    print(title)
    if cnt:
        for r, x in m:
            avg = x / cnt[r]
            print(x, avg, r)
    else:
        for r, x in m:
            print(x, r)
    print()


def main():
    reqtimes = defaultdict(int)
    req_out = defaultdict(int)
    req_count = defaultdict(int)

    for L in fileinput.input():
        request = reqtime = outbytes = None
        cs = L.split('\t')
        for c in cs:
            if c.startswith('reqtime:'):
                reqtime = float(c.split(':', 1)[1])
            if c.startswith('out:'):
                outbytes = int(c.split(':', 1)[1])
            if c.startswith('request:'):
                request = c.split(':', 1)[1]
                request = ' '.join(request.split(' ')[:2])
        if not request:
            continue

        # ここで、アクセスのパターンをまとめる。
        # いかは ISUCON3 の時の例
        #if request.startswith('GET /memo/'):  # /memo/12345 のようなリクエストを /memo/* にする
        #    request = 'GET /memo/*'
        #if request.startswith('GET /recent/'):
        #    request = 'GET /recent/*'

        req_count[request] += 1
        if reqtime:
            reqtimes[request] += reqtime
        if outbytes:
            req_out[request] += outbytes

    summary_map("Request by count", req_count)
    summary_map("Request by total time", reqtimes, req_count)
    summary_map("Request by out bytes", req_out, req_count)

if __name__ == '__main__':
    main()
