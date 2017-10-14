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
