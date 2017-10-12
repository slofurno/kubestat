function pack(r,g,b,a) {
    return (a << 24) | (b << 16) | (g << 8) | r
}

const scale_factor = 8

function newHeatmap(root, {width, height, onclick, scaleY, scaleX}) {

    let series = []
    let start_ = 0

    let buffer = new ArrayBuffer(width * height * 4)
    let writeView = new Uint32Array(buffer)
    let readView = new Uint8ClampedArray(buffer)
    let imageData = new ImageData(readView, width, height)

    let canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    let ctx = canvas.getContext('2d')
    ctx.strokeStyle = 'white'

    function render() {
        ctx.putImageData(imageData, 0, 0)
    }

    render()

    canvas.style.width = scale_factor * width + "px"
    canvas.style.height = scale_factor * height + "px"

    canvas.onclick = function(e) {
        let rect = e.target.getBoundingClientRect();
        let x = e.clientX - rect.left
        let y = e.clientY - rect.top

        if (typeof onclick === 'function') {
            onclick(x/scale_factor|0, y/scale_factor|0)
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
        let cols = []

        for (let j = 0; j < width; j++) {
          let z = []
          for (let i = 0; i < height; i++) { z.push(0) }
          cols.push(z)
        }

        let next = []
        let end = start_ + scaleX * width

        for (let i = 0; i < series.length; i++) {
          let t = series[i][0]
          if (t < start_) { continue }

          next.push(series[i])

          if (t < end) {
            let h = Math.min((series[i][1]/scaleY)|0, height-1)
            cols[((t-start_)/scaleX)|0][h]++
          }
        }

        let ranks = mergeranks(cols)
        for (let i = 0; i < cols.length; i++){
            let col = cols[i]
            for(let j = 0; j < col.length; j++){
                let k = (height - 1 - j) * width + i
                writeView[k] = toColor(ranks[col[j]])
            }
        }

        series = next
        ctx.putImageData(imageData, 0, 0)
    }

    function setTime(time) {
      start_ = time - width * scaleX
    }

    function getSeries() {
      return series
    }

    return {writeView, canvas, destroy, push_back, setTime, getSeries}

}

function mergeranks(xxs) {
    let seen = {}
    let ys = []
    xxs.forEach(xs =>{
        xs.forEach(x => {
            if (!seen[x]) {
                seen[x] = 1
                ys.push(x)
            }
        })
    })

    ys.sort()
    ys.forEach((y,i) => seen[y] = i / ys.length)
    return seen
}

function makeFakeSource(n) {
    let pods = []

    for (let i = 0; i<n; i++) {
        let period = Math.random() * 20000 + 2000
        let seed = Math.random()
        let A = Math.random() * 0.1 + (seed > 0.8 ? 0.7 : 0.2)
        let cpu = 0

        pods.push({id: i, A, period, cpu, phase: seed * Math.PI * 2, base: seed})
    }

    function next() {
        let time = (new Date()).getTime()
        let ret = []
        let avail = 8
        let need = 0

        for (let i = 0; i < n; i++) {
            let p = pods[i]
            p.cpu += 0.5 * (p.A + p.A * Math.cos(time/p.period + p.phase))
            need += p.cpu
        }

        for (let i = 0; i < n; i++) {
            let p = pods[i]

            let cpu = need > avail ? p.cpu * (avail/need) : p.cpu
            p.cpu -= Math.min(0.95, cpu)
            let t = time + (Math.random() * 50 - 25 | 0)

            ret.push({cpu, throttled_time: p.cpu, t})
        }

        return ret
    }

    return {
        next,
    }
}

function toColor(percent) {
    return hslToRgb(0, 0.95 * percent, 120/255)
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

    //return [r * 255, g * 255, b * 255];
    return pack(r*255, g*255, b*255, 255)
}

//ws:///${location.host}/ws

