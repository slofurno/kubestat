function pack(r,g,b,a) {
    return (a << 24) | (b << 16) | (g << 8) | r
}

const scale_factor = 10

const colorspaces = {
    "LCH": toColorLCH,
    "HSL": toColorHSL,
}

function newHeatmap(root, {width, height, onclick, scaleY, scaleX, colorspace}) {

    const toColor = colorspaces[colorspace || "HSL"]
    let series = []

    let buffer = new ArrayBuffer(width * height * 4)
    let writeView = new Uint32Array(buffer)
    let readView = new Uint8ClampedArray(buffer)
    let imageData = new ImageData(readView, width, height)

    let canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    let ctx = canvas.getContext('2d')
    ctx.strokeStyle = 'white'

    let cols = []
    for (let j = 0; j < width; j++) {
        let z = []
        for (let i = 0; i < height; i++) { z.push([]) }
        cols.push(z)
    }

    canvas.style.width = scale_factor * width + "px"
    canvas.style.height = scale_factor * height + "px"

    canvas.onclick = function(e) {
        let rect = e.target.getBoundingClientRect();
        let screenX = e.clientX - rect.left
        let screenY = e.clientY - rect.top

        let x = screenX/scale_factor|0
        let y = screenY/scale_factor|0


        if (typeof onclick === 'function') {
            onclick(cols[x][height - 1 - y], screenX, screenY)
        }
    }

    root.appendChild(canvas)

    function destroy() {
        canvas.onclick = null
        root.innerHTML = ""
    }

    function push_back(xs) {
        for (let i = 0; i < xs.length; i++) { series.push(xs[i]) }
        render_()
    }

    function render_() {
        let t0 = performance.now()
        let start = (((new Date()).getTime() - width * scaleX)/scaleX|0)*scaleX
        let end = start + scaleX * width
        let next = []

        for (let i = 0; i < width; i++) {
            for (let j = 0; j < height; j++) {
                cols[i][j] = []
            }
        }

        for (let i = 0; i < series.length; i++) {
          let t = series[i][0]
          if (t < start) { continue }

          next.push(series[i])

          if (t < end) {
            let h = Math.min((series[i][1]/scaleY)|0, height-1)
            cols[((t-start)/scaleX)|0][h].push(series[i])
          }
        }

        //let ranks = rankedsaturation(cols)
        for (let i = 0; i < width; i++){
            let col = cols[i]
            let ranks = ranksaturation(col)

            for(let j = 0; j < height; j++){
                let k = (height - 1 - j) * width + i
                writeView[k] = toColor(ranks[col[j].length])
            }
        }

        series = next
        ctx.putImageData(imageData, 0, 0)
    }

    return {writeView, canvas, destroy, push_back }
}

function rankedsaturation(xxs) {
    let seen = {0: -1}
    let ys = []
    xxs.forEach(xs =>{
        xs.forEach(x => {
            let k = x.length
            if (!seen[k]) {
                seen[k] = 1
                ys.push(k)
            }
        })
    })

    ys.sort((a,b) => a-b)
    ys.forEach((y,i) => seen[y] = i / (ys.length-1))
    return seen
}

function ranksaturation(xs) {
    let seen = {0: -1}
    let ys = []
    xs.forEach(x => {
        let k = x.length
        if (!seen[k]) {
            seen[k] = 1
            ys.push(k)
        }
    })

    ys.sort((a,b) => a-b)
    ys.forEach((y,i) => seen[y] = i / (ys.length-1))
    return seen
}

function linearsaturation(xxs) {
    let seen = {}
    let max = 0
    for (let j = 0; j < xxs.length; j++){
        for(let i = 0; i < xxs[j].length; i++) {
            let k = xxs[j][i].length
            seen[k] = 1
            max = k > max ? k : max
        }
    }

    Object.keys(seen).forEach(x => seen[x] = x/max)
    return seen
}


const offwhite = pack(245, 245, 245, 255)
//53.24079414130722, 104.55176567686985, 39.99901061253297

function toColorHSL(percent) {
    return percent === -1 ? offwhite : hslToRgb(0, percent, 0.6)
}

function toColorLCH(percent) {
    return percent === -1 ? offwhite : lch2rgb(53.24, percent * 104, 40)
}

function hslToRgb(h, s, l){
    var r, g, b;

    if(s == 0){
        r = g = b = l; // achromatic
    }else{
        function hue2rgb(p, q, t){
            if(t < 0) t += 1;
            if(t > 1) t -= 1;
            if(t < 1/6) return p + (q - p) * 6 * t;
            if(t < 1/2) return q;
            if(t < 2/3) return p + (q - p) * (2/3 - t) * 6;
            return p;
        }

        var q = l < 0.5 ? l * (1 + s) : l + s - l * s;
        var p = 2 * l - q;
        r = hue2rgb(p, q, h + 1/3);
        g = hue2rgb(p, q, h);
        b = hue2rgb(p, q, h - 1/3);
    }

    return pack(r*255, g*255, b*255, 255)
}

const DEG2RAD = Math.PI / 180;
const pow = Math.pow
const cos = Math.cos
const sin = Math.sin
const sqrt = Math.sqrt
const atan2 = Math.atan2
const RAD2DEG = 180 / Math.PI;
const round = Math.round

const LAB_CONSTANTS = {
    Kn: 18,
    Xn: 0.950470,
    Yn: 1,
    Zn: 1.088830,
    t0: 0.137931034,
    t1: 0.206896552,
    t2: 0.12841855,
    t3: 0.008856452
  };

function lab_xyz(t) {
    if (t > LAB_CONSTANTS.t1) {
      return t * t * t;
    } else {
      return LAB_CONSTANTS.t2 * (t - LAB_CONSTANTS.t0);
    }
  }

function unpack (args) {
    if (args.length >= 3) {
      return [].slice.call(args);
    } else {
      return args[0];
    }
  }

function lch2rgb () {
    var L, a, args, b, c, g, h, l, r, ref, ref1;
    args = unpack(arguments);
    l = args[0], c = args[1], h = args[2];
    ref = lch2lab(l, c, h), L = ref[0], a = ref[1], b = ref[2];
    ref1 = lab2rgb(L, a, b), r = ref1[0], g = ref1[1], b = ref1[2];
    return pack(r,g,b,255)
    //return [r, g, b, 255];
  }

  
  function lch2lab() {
    
        /*
        Convert from a qualitative parameter h and a quantitative parameter l to a 24-bit pixel.
        These formulas were invented by David Dalrymple to obtain maximum contrast without going
        out of gamut if the parameters are in the range 0-1.
        
        A saturation multiplier was added by Gregor Aisch
         */
        var c, h, l, ref;
        ref = unpack(arguments), l = ref[0], c = ref[1], h = ref[2];
        h = h * DEG2RAD;
        return [l, cos(h) * c, sin(h) * c];
      };

function lab2rgb() {
        var a, args, b, g, l, r, x, y, z;
        args = unpack(arguments);
        l = args[0], a = args[1], b = args[2];
        y = (l + 16) / 116;
        x = isNaN(a) ? y : y + a / 500;
        z = isNaN(b) ? y : y - b / 200;
        y = LAB_CONSTANTS.Yn * lab_xyz(y);
        x = LAB_CONSTANTS.Xn * lab_xyz(x);
        z = LAB_CONSTANTS.Zn * lab_xyz(z);
        r = xyz_rgb(3.2404542 * x - 1.5371385 * y - 0.4985314 * z);
        g = xyz_rgb(-0.9692660 * x + 1.8760108 * y + 0.0415560 * z);
        b = xyz_rgb(0.0556434 * x - 0.2040259 * y + 1.0572252 * z);
        return [r, g, b, args.length > 3 ? args[3] : 1];
      };

function xyz_rgb (r) {
    return 255 * (r <= 0.00304 ? 12.92 * r : 1.055 * pow(r, 1 / 2.4) - 0.055);
}

function rgb2lch() {
    var a, b, g, l, r, ref, ref1;
    ref = unpack(arguments), r = ref[0], g = ref[1], b = ref[2];
    ref1 = rgb2lab(r, g, b), l = ref1[0], a = ref1[1], b = ref1[2];
    return lab2lch(l, a, b);
  };

  function rgb2lab () {
    var b, g, r, ref, ref1, x, y, z;
    ref = unpack(arguments), r = ref[0], g = ref[1], b = ref[2];
    ref1 = rgb2xyz(r, g, b), x = ref1[0], y = ref1[1], z = ref1[2];
    return [116 * y - 16, 500 * (x - y), 200 * (y - z)];
  };

function rgb2xyz () {
    var b, g, r, ref, x, y, z;
    ref = unpack(arguments), r = ref[0], g = ref[1], b = ref[2];
    r = rgb_xyz(r);
    g = rgb_xyz(g);
    b = rgb_xyz(b);
    x = xyz_lab((0.4124564 * r + 0.3575761 * g + 0.1804375 * b) / LAB_CONSTANTS.Xn);
    y = xyz_lab((0.2126729 * r + 0.7151522 * g + 0.0721750 * b) / LAB_CONSTANTS.Yn);
    z = xyz_lab((0.0193339 * r + 0.1191920 * g + 0.9503041 * b) / LAB_CONSTANTS.Zn);
    return [x, y, z];
  };

 function rgb_xyz (r) {
    if ((r /= 255) <= 0.04045) {
      return r / 12.92;
    } else {
      return pow((r + 0.055) / 1.055, 2.4);
    }
  };

  function xyz_lab(t) {
    if (t > LAB_CONSTANTS.t3) {
      return pow(t, 1 / 3);
    } else {
      return t / LAB_CONSTANTS.t2 + LAB_CONSTANTS.t0;
    }
  };

  function lab2lch() {
    var a, b, c, h, l, ref;
    ref = unpack(arguments), l = ref[0], a = ref[1], b = ref[2];
    c = sqrt(a * a + b * b);
    h = (atan2(b, a) * RAD2DEG + 360) % 360;
    if (round(c * 10000) === 0) {
      h = Number.NaN;
    }
    return [l, c, h];
  };