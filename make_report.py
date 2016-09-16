import csv
import os
import sys
import cgi

def print_mysql_result(out):
    out.write("<h2>MySQL Slow Queries</h2>")
    if not os.path.exists('mysql.txt'):
        out.write("<p>mysql.txt not found.</p>")
        return
    with open('mysql.txt') as f:
        out.write("""<pre class="prettyprint">""")
        for line in f.readlines():
            out.write(cgi.escape(line))
        out.write("</pre>")

def print_myprofiler_result(out):
    out.write("<h2>myprofiler</h2>")
    if not os.path.exists('myprofiler.log'):
        out.write("<p>myprofiler.log not found.</p>")
        return
    with open('myprofiler.log') as f:
        out.write("""<pre class="prettyprint">""")
        lines = f.readlines()
        idx = 0
        for i in xrange(len(lines)):
            line = lines[i]
            if line[0].startswith("#"):
                idx = i
        for line in lines[idx:]:
            out.write(cgi.escape(line))
        out.write("</pre>")

def print_bench_result(out):
    out.write("<h2>Bench Output</h2>")
    if not os.path.exists('bench.txt'):
        out.write("<p>bench.txt not found.</p>")
        return
    with open('bench.txt') as f:
        out.write("""<pre class="prettyprint">""")
        for line in f.readlines():
            out.write(cgi.escape(line))
        out.write("</pre>")

def print_summary(out):
    out.write("<h2>Request Summary</h2>")
    if not os.path.exists('summary.txt'):
        out.write("<p>summary.txt not found.</p>")
        return
    with open('summary.txt') as f:
        opened = False
        for line in f.readlines():
            if line.lstrip().startswith("Request by"):
                if opened:
                    out.write("</small></pre>")
                sys.stdout.write("""<pre class="col-md-4 prettyprint lang-sh"><small>""")
                out.write(cgi.escape(line))
                opened = True
            else:
                out.write(cgi.escape(line))
        if opened:
            out.write("</small></pre>")

def parse_dstat_report(out):
    if not os.path.exists('dstat.csv'):
        out.write("<p>dstat.csv not found.</p>")
        return

    kind = []
    header = []
    data = []
    with open('dstat.csv') as f:
        r = csv.reader(f)
        kind = next(r)
        for i in xrange(10):
            if kind and kind[0] == 'system':
                break
            kind = next(r)
        else:
            out.write('failed to parse dstat file.')
            return
        header = next(r)
        data = [row for row in r]

    n = len(header)
    h = len(data)

    def find_kind_index(name):
        begin = end = -1
        for i in xrange(n):
            if begin == -1 and kind[i].find(name) != -1:
                begin = i
            elif begin != -1 and (kind[i] or i == n-1):
                end = i
                if i == n-1:
                    end = n
                break
        return (begin, end, begin != -1 and end != -1)

    def make_csv(name, loffset=0, roffset=0):
        start, end, ok = find_kind_index(name)
        if not ok:
            return None
        start += loffset
        end += roffset
        ret = []
        ret.append(header[:1] + header[start:end])
        for i in xrange(h):
            ret.append(data[i][:1] + map(float, data[i][start:end]))
        return ret

    return make_csv('cpu', 0, -2), make_csv('mem'), make_csv('net')

def print_dstat_report(out):
    cpu_csv, mem_csv, net_csv = parse_dstat_report(out)

    out.write(u"""
      <script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);
      function drawChart() {
        var charts = document.getElementById('charts')""")

    if cpu_csv:
        out.write("""
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable([""")
        for line in cpu_csv:
            out.write("%r,\n" % line)
        out.write("""]), {
                title: 'cpu',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0},
	        legend: {position: 'top', maxLines: 3},
                backgroundColor: { fill:'transparent' },
                isStacked: true,
            });""")

    if mem_csv:
        out.write("""
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable([""")
        for line in mem_csv:
            out.write("%r,\n" % line)
        out.write("""]), {
                title: 'mem',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0, format: 'short'},
	        legend: {position: 'top', maxLines: 3},
                backgroundColor: { fill:'transparent' },
            });""")

    if net_csv:
        out.write("""
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable([""")
        for line in net_csv:
            out.write("%r,\n" % line)
        out.write("""]), {
                title: 'net',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0},
	        legend: {position: 'top', maxLines: 3},
                backgroundColor: { fill:'transparent' },
            });""")

    out.write("""
     }
     </script>
    """)

def parse_pidstat_report(out):
    if not os.path.exists('pidstat.csv'):
        out.write("<p>pidstat.csv not found.</p>")
        return None, None

    table = {}
    cmds = set()
    with open('pidstat.csv') as f:
        r = csv.reader(f)
        for line in r:
            t = line[0]
            cmd = line[-1]
            if t not in table:
                table[t] = {}
            cmds.add(cmd)
            table[t][cmd] = (line[6], line[-2])

    cmds = sorted(list(cmds))

    cpu = []
    cpu.append(["Time"] + cmds)

    mem = []
    mem.append(["Time"] + cmds)

    for t in table:
        line_cpu = [t]
        line_mem = [t]
        for cmd in cmds:
            if cmd in table[t]:
                line_cpu.append(float(table[t][cmd][0]))
                line_mem.append(float(table[t][cmd][1]))
            else:
                line_cpu.append(0)
                line_mem.append(0)
        cpu.append(line_cpu)
        mem.append(line_mem)
    return cpu, mem

def parse_netstat_result(out):
    if not os.path.exists('netstat.log'):
        out.write("<p>netstat.log not found.</p>")
        return

    sep = []
    states = set()
    with open('netstat.log') as f:
        counter = {}
        for line in f:
            s = line.strip().split()
            if s[0].startswith('tcp'):
                if s[-1].isupper():
                    states.add(s[-1])
                    if s[-1] not in counter:
                        counter[s[-1]] = 0
                    counter[s[-1]] += 1
            elif counter:
                sep.append(counter)
                counter = {}

    states = sorted(list(states))
    ret = []
    ret.append(["Seq"] + states)

    i = 0
    for counter in sep:
        line = [i]
        for s in states:
            if s in counter:
                line.append(counter[s])
            else:
                line.append(0)
        i += 1
        ret.append(line)
    return ret

def print_pidstat_report(out):
    cpu_csv, mem_csv = parse_pidstat_report(out)
    tcp_csv = parse_netstat_result(out)

    out.write(u"""
      <script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);
      function drawChart() {
        var charts = document.getElementById('charts')""")

    if cpu_csv:
        out.write("""
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable([""")
        for line in cpu_csv:
            out.write("%r,\n" % line)
        out.write("""]), {
                title: '%cpu',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0},
	        legend: {position: 'top', maxLines: 3},
                backgroundColor: { fill:'transparent' },
                isStacked: true,
            });""")

    if mem_csv:
        out.write("""
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable([""")
        for line in mem_csv:
            out.write("%r,\n" % line)
        out.write("""]), {
                title: '%mem',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0},
	        legend: {position: 'top', maxLines: 3},
                backgroundColor: { fill:'transparent' },
                isStacked: true,
            });""")

    if tcp_csv:
        out.write("""
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable([""")
        for line in tcp_csv:
            out.write("%r,\n" % line)
        out.write("""]), {
                title: 'tcp',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0, format: 'short'},
	        legend: {position: 'top', maxLines: 3},
                backgroundColor: { fill:'transparent' },
            });""")

    out.write("""
     }
     </script>
    """)

def split_by_rountine(lines):
    reports = []
    title = ""
    value = 0
    content = []
    for line in lines:
        if line.startswith("ROUTINE ====="):
            if content:
                reports.append((value, title, content))
            title = line.split()[2]
            content = []
            value = -1
        elif line.strip().endswith("% of Total"):
            value = float(line.split()[-3][:-1])
        else:
            content.append(line)
    if content:
        reports.append((value, title, content))
    reports.sort(key=lambda x: x[0], reverse=True)
    return reports

def print_app_profiles(out):
    out.write("""<h2>CPU Profile (<a href="cpu.svg">pprof SVG</a>) (<a href="cpu-torch.svg">FlameGraph SVG</a>)</h2>""")
    with open('cpu.txt') as f:
        lines = f.readlines()
        out.write("""<pre class="prettyprint">""")
        for line in lines[:min(20, len(lines))]:
            out.write(line)
        out.write("</pre>")
    with open('cpu-cum.txt') as f:
        lines = f.readlines()
        out.write("""<pre class="prettyprint">""")
        for line in lines[:min(20, len(lines))]:
            out.write(line)
        out.write("</pre>")
    with open('cpu.list') as f:
        r = split_by_rountine(f)
        for rountine in r[:min(5, len(r))]:
            out.write("<h3>[CPU] %s%% %s</h3>" % (rountine[0], rountine[1]))
            out.write("""<pre class="prettyprint">""")
            for line in rountine[2]:
                out.write(cgi.escape(line[10:]))
            out.write("</pre>")

    out.write("""<h2>MEM Profile (<a href="mem.svg">pprof SVG</a>) (<a href="mem-torch.svg">FlameGraph SVG</a>)</h2>""")
    with open('mem.txt') as f:
        lines = f.readlines()
        out.write("""<pre class="prettyprint">""")
        for line in lines[:min(20, len(lines))]:
            out.write(line)
        out.write("</pre>")
    with open('mem-cum.txt') as f:
        lines = f.readlines()
        out.write("""<pre class="prettyprint">""")
        for line in lines[:min(20, len(lines))]:
            out.write(line)
        out.write("</pre>")
    with open('mem.list') as f:
        r = split_by_rountine(f)
        for rountine in r[:min(3, len(r))]:
            out.write("<h3>[MEM] %s%% %s</h3>" % (rountine[0], rountine[1]))
            out.write("""<pre class="prettyprint">""")
            for line in rountine[2]:
                out.write(cgi.escape(line[10:]))
            out.write("</pre>")

    out.write("""<h2>BLOCK Profile (<a href="block.svg">pprof SVG</a>) (<a href="block-torch.svg">FlameGraph SVG</a>)</h2>""")
    with open('block.txt') as f:
        lines = f.readlines()
        out.write("""<pre class="prettyprint">""")
        for line in lines[:min(20, len(lines))]:
            out.write(line)
        out.write("</pre>")
    with open('block-cum.txt') as f:
        lines = f.readlines()
        out.write("""<pre class="prettyprint">""")
        for line in lines[:min(20, len(lines))]:
            out.write(line)
        out.write("</pre>")
    with open('block.list') as f:
        r = split_by_rountine(f)
        for rountine in r[:min(3, len(r))]:
            out.write("<h3>[BLOCK] %s%% %s</h3>" % (rountine[0], rountine[1]))
            out.write("""<pre class="prettyprint">""")
            for line in rountine[2]:
                out.write(cgi.escape(line[10:]))
            out.write("</pre>")

def main():
    os.chdir(sys.argv[1])
    out = sys.stdout
    out.write(u"""
<!DOCTYPE HTML>
<html>
  <head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.12.4/jquery.min.js"></script>
  <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js" integrity="sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa" crossorigin="anonymous"></script>
  <script src="https://cdn.rawgit.com/google/code-prettify/master/loader/run_prettify.js?skin=desert"></script>
  <script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
  <style></style>
""")
    print_dstat_report(out)
    print_pidstat_report(out)
    out.write(u"""
  </head>
  <body>
  <div class="container">
	<h1>Benchmark Report</h1>
	<div id="charts" class="row"></div>
""")
    print_bench_result(out)
    print_summary(out)
    print_mysql_result(out)
    print_myprofiler_result(out)
    print_app_profiles(out)
    out.write(u"""
  </div>
  </body>
</html>
""")

if __name__ == '__main__':
    main()
