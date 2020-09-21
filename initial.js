var data = [
    {{- range $i, $ts := .Ordered }}
    {{- if (ne $i 0) }},{{end}}
    [new Date({{$ts -}})
        {{- range (index $.Points . ) -}}, 
            {{- if (or (eq . 0.0) (eq . -3600e3)) -}}
                {{- -100.0 -}}
            {{- else -}}
                {{- . -}}
            {{end -}}
        {{end -}}
    ]
    {{- end}}
];
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
        }
    }
);
