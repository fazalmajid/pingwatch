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
var autoscroll = document.forms[0].autoscroll;
if (data.length > 0) {
    dmin = data[0][0].getTime();
    dmax = data[data.length-1][0].getTime();
}
function hsvToRGB(hue, saturation, value) {
  var red;
  var green;
  var blue;
  if (saturation === 0) {
    red = value;
    green = value;
    blue = value;
  } else {
    var i = Math.floor(hue * 6);
    var f = (hue * 6) - i;
    var p = value * (1 - saturation);
    var q = value * (1 - (saturation * f));
    var t = value * (1 - (saturation * (1 - f)));
    switch (i) {
      case 1: red = q; green = value; blue = p; break;
      case 2: red = p; green = value; blue = t; break;
      case 3: red = p; green = q; blue = value; break;
      case 4: red = t; green = p; blue = value; break;
      case 5: red = value; green = p; blue = q; break;
      case 6: // fall through
      case 0: red = value; green = t; blue = p; break;
    }
  }
  red = Math.floor(255 * red + 0.5);
  green = Math.floor(255 * green + 0.5);
  blue = Math.floor(255 * blue + 0.5);
  return 'rgb(' + red + ',' + green + ',' + blue + ')';
};
function get_color(i, total) {
    // use the IBM Design Language color palette as seen at:
    // https://www.carbondesignsystem.com/data-visualization/color-palettes/
    //
    // best would be to use the algorithm https://arxiv.org/pdf/2009.02969.pdf
    if (total <= 5) {
        return [
            ["#6929c4"],
            ["#6929c4", "#009d9a"],
            ["#a56eff", "#005d5d", "#9f1853"],
            ["#6929c4", "#012749", "#009d9a", "#ee538b"],
            ["#6929c4", "#1192e8", "#005d5d", "#9f1853", "#570408"],
        ][total-1][i]
    }
    if (total <= 14) {
        return [
            "#6929c4", "#1192e8", "#005d5d", "#9f1853", "#fa4d56", "#570408",
            "#198038", "#002d9c", "#ee538b", "#b28600", "#009d9a", "#012749",
            "#8a3800", "#a56eff"
        ][i]
    }
    // excessive number of colors, try a spiral
    return hsvToRGB(i/8 - Math.floor(i/8), 1.0, 0.75 - 0.5*(i/total));
}
{{ $host_count := len .Header }}
g = new Dygraph(
    document.getElementById("ping"), data, {
        labels: [{{range $i, $h := .Header -}}
            {{if (ne $i 0)}},{{end}}"{{$h}}"
        {{- end}}],
        colors: [{{range $i, $h := .Header -}}
                 {{if (ne $i 0)}},{{end}}get_color({{- $i -}}, {{- $host_count -}} - 1)
        {{- end}}],
        title: "Ping time (ms)",
        highlightCircleSize: 2,
        strokeWidth: 1,
        strokeBorderWidth: 1,
        //colorValue: 0.75,
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
            let last = data[data.length-1][0].getTime();
            if (dmax == 0 || dmax >= last - 300000) {
                autoscroll.checked = true;
            } else {
                autoscroll.checked = false;
            }
        }
    }
);

function update() {
    // only autoupdate if the current zoom selection goes all the way to the
    // latest known data
    let last = data[data.length-1][0].getTime();
    // add a fudge factor of up to 5 minutes
    if (autoscroll.checked) {
        document.body.removeChild(document.getElementById("update"));
        script = document.createElement("SCRIPT");
        script.src = "/delta?since=" + String(last);
        script.id = "update";
        document.body.appendChild(script);
    }
}

window.setInterval(update, {{$.Delay}});
