function pack(r,g,b,a) {
    return (a << 24) | (b << 16) | (g << 8) | r
}

const scale_factor = 10

function newHeatmap(root, {width, height, onclick, scaleY, scaleX, historical}) {

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
        let start = historical
          ? minimumTime(series)
          : (((new Date()).getTime() - width * scaleX)/scaleX|0)*scaleX

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
    console.log(ys, seen)
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

function toColor(percent) {
    return percent === -1 ? offwhite : hslToRgb(0, percent, 0.6)
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

function minimumTime(xs) {
  let t0 = Date.now()
  for (let i = 0; i < xs.length; i++) {
    if (xs[i][0] < t0) {
      t0 = xs[i][0]
    }
  }
  return t0
}
