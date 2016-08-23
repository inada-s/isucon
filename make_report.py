import csv
import os
import sys

def print_mysql_result(out):
    out.write("<h2>MySQL Slow Queries</h2>")
    if not os.path.exists('mysql.txt'):
        out.write("<p>mysql.txt not found.</p>")
        return
    with open('mysql.txt') as f:
        out.write("""<pre class="prettyprint">""")
        for line in f.readlines():
            out.write(line)
        out.write("</pre>")

def print_bench_result(out):
    out.write("<h2>Bench Output</h2>")
    if not os.path.exists('bench.txt'):
        out.write("<p>bench.txt not found.</p>")
        return
    with open('bench.txt') as f:
        out.write("""<pre class="prettyprint">""")
        for line in f.readlines():
            out.write(line)
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
                out.write(line)
                opened = True
            else:
                out.write(line)
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
    out.write("""<h2>CPU Profile (<a href="cpu.svg">SVG</a>)</h2>""")
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
        for rountine in r[:min(3, len(r))]:
            out.write("<h3>[CPU] %s%% %s</h3>" % (rountine[0], rountine[1]))
            out.write("""<pre class="prettyprint">""")
            for line in rountine[2]:
                out.write(line[10:])
            out.write("</pre>")

    out.write("""<h2>MEM Profile (<a href="mem.svg">SVG</a>)</h2>""")
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
                out.write(line[10:])
            out.write("</pre>")

    out.write("""<h2>BLOCK Profile (<a href="block.svg">SVG</a>)</h2>""")
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
                out.write(line[10:])
            out.write("</pre>")

def main():
    os.chdir(sys.argv[1])
    out = sys.stdout
    out.write(u"""
<!DOCTYPE HTML>
<html>
  <head>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.12.4/jquery.min.js"></script>
  <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js" integrity="sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa" crossorigin="anonymous"></script>
  <script src="https://cdn.rawgit.com/google/code-prettify/master/loader/run_prettify.js?skin=desert"></script>
  <script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
  <style></style>
""")
    print_dstat_report(out)
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
    print_app_profiles(out)
    out.write(u"""
  </div>
  </body>
</html>
""")

if __name__ == '__main__':
    main()
