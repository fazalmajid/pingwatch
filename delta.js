var delta = [
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

if (delta.length > 0) {
    let dlen = data.length;
    let first = delta[0][0];
    // two updates may have collided, get rid of overlapping data
    while (data.length > 0 && data[data.length-1][0] > first) {
        data.pop();
    }
    // add the new data at the end
    Array.prototype.push.apply(data, delta);
    // remove older data until the size of the data window remains the same
    // but only once we've met the minimum number of data points, otherwise
    // when we first get started the window would remain less than the
    // desired window
    while (data.length > dlen && data.length > {{$.MinData}}) {
        data.shift();
    }
    if (dmin == 0) {
        // there was no initial data, delta is the entire new data
        dmin = delta[0][0].getTime();
        dmax = delta[delta.length-1][0].getTime();
    } else {
        let new_dmax = delta[delta.length-1][0].getTime();
        dmin = dmin + (new_dmax - dmax);
        dmax = new_dmax;
    }
    rowmax = data.length - 1;
    delta = null;
    g.updateOptions({
        "file": data,
        "dateWindow": [dmin, dmax]
    });
}
