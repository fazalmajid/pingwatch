var data = [
    {{- range $i, $ts := .Ordered }}
    {{- if (ne $i 0) }},{{end}}
    [new Date({{$ts -}})
        {{- range (index $.Points $ts ) -}}, 
            {{- if (or (eq . 0.0) (eq . -3600e3)) -}}
                {{- -100.0 -}}
            {{- else -}}
                {{- . -}}
            {{end -}}
        {{end -}}
    ]
    {{- end}}
];
var dmin = 0;
var dmax = 0;
if (data.length > 0) {
    dmin = data[0][0].getTime();
    dmax = data[data.length-1][0].getTime();
}
g = new Dygraph(
    document.getElementById("ping"), data, {
        labels: [{{range $i, $h := .Header -}}
            {{if (ne $i 0)}},{{end}}"{{$h}}"
        {{- end}}],
        title: "Ping time (ms)",
        width: window.innerWidth * 0.95,
        height: window.innerHeight / 2,
        highlightCircleSize: 2,
        strokeWidth: 1,
        strokeBorderWidth: 1,
        legend: "always",
        labelsDiv: "legend",
        labelsSeparateLines: true,
        highlightSeriesOpts: {
            strokeWidth: 3,
            strokeBorderWidth: 1,
            highlightCircleSize: 5,
        },
        zoomCallback: function(minDate, maxDate, yRanges) {
            dmin = minDate;
            dmax = maxDate;
        }
    }
);

function update() {
    // only autoupdate if the current zoom selection goes all the way to the
    // latest known data
    let last = data[data.length-1][0].getTime();
    // add a fudge factor of up to 5 minutes
    if (dmax == 0 || dmax >= last - 300000) {
        document.body.removeChild(document.getElementById("update"));
        script = document.createElement("SCRIPT");
        script.src = "/delta?since=" + String(last);
        script.id = "update";
        document.body.appendChild(script);
    }
}

window.setInterval(update, {{$.Delay}});
