import csv
import os
import StringIO

dstat_csv_path = '/tmp/dstat.csv'

def dstat_report():
    if not os.path.exists(dstat_csv_path):
        print 'dstat file not found.'
        return

    kind = []
    header = []
    data = []
    with open(dstat_csv_path) as f:
        r = csv.reader(f)
        kind = next(r)
        for i in xrange(10):
            if kind and kind[0] == 'system':
                break
            kind = next(r)
        else:
            print 'failed to parse dstat file.'
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

def main():
    cpu_csv, mem_csv, net_csv = dstat_report()

    print u"""
    <!DOCTYPE HTML>
    <html>
      <head>
      <meta name="viewport" content="width=device-width, initial-scale=1">
      <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
      <script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
      <script type="text/javascript">
      google.charts.load('current', {'packages':['corechart']});
      google.charts.setOnLoadCallback(drawChart);
      function drawChart() {
        var charts = document.getElementById('charts')"""

    if cpu_csv:
        print """
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable(["""
        for line in cpu_csv:
            print line, ","
        print """]), {
                title: 'cpu',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0},
	        legend: {position: 'top', maxLines: 3},
            });"""

    if mem_csv:
        print """
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable(["""
        for line in mem_csv:
            print line, ","
        print """]), {
                title: 'mem',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0, format: 'short'},
	        legend: {position: 'top', maxLines: 3},
            });
        """

    if net_csv:
        print """
            var div = document.createElement("div");
            div.setAttribute("class", "col-md-4")
            charts.appendChild(div);
            var chart = new google.visualization.AreaChart(div);
            chart.draw(google.visualization.arrayToDataTable(["""
        for line in net_csv:
            print line, ","
        print """]), {
                title: 'net',
	        width: 400,
	        height: 240,
                vAxis: {minValue: 0},
	        legend: {position: 'top', maxLines: 3},
            });
        """

    print """
     }
    """

    print u"""
    </script>
      </head>
      <body>
        <div class="container">
        <h1>Benchmark Report</h1>
	<div id="charts" class="row"></div>
        </div>
      </body>
      <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.12.4/jquery.min.js"></script>
      <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js" integrity="sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa" crossorigin="anonymous"></script>
    </html>
    """

if __name__ == '__main__':
    main()
