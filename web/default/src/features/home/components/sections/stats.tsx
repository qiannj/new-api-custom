import { useRef, useEffect, useCallback } from "react"
import { useTranslation } from "react-i18next"

function Counter(props: {end:number; suffix?:string}) {
  const ref = useRef<HTMLSpanElement>(null); const s = useRef(false)
  const animate = useCallback(() => {
    const el = ref.current; if (!el) return; const start = performance.now()
    const step = (now:number) => {
      const p = Math.min((now-start)/1600, 1)
      el.textContent = `${Math.round(Math.pow(p,0.5)*props.end)}${props.suffix||""}`
      if (p < 1) requestAnimationFrame(step)
    }; requestAnimationFrame(step)
  }, [props.end, props.suffix])
  useEffect(() => {
    const el = ref.current; if (!el) return
    const o = new IntersectionObserver(([e]) => { if (e.isIntersecting && !s.current) { s.current=true; animate(); o.unobserve(el) } }, {threshold:0.5})
    o.observe(el); return () => o.disconnect()
  }, [animate])
  return <span ref={ref} className="tabular-nums">0{props.suffix||""}</span>
}

export function Stats() {
  const {t} = useTranslation()
  const items = [
    {end:7, suffix:"+", label:t("Available Models")},
    {end:24, suffix:"/7", label:t("24/7 Service")},
    {end:100, suffix:"+", label:t("Countries Accessible")},
    {end:99.9, suffix:"%", label:t("Service Reliability")},
  ]
  return (
    <div className="border-border/40 bg-muted/10 relative z-10 border-y">
      <div className="mx-auto max-w-6xl px-6 py-10 md:py-12">
        <div className="grid grid-cols-2 gap-8 md:grid-cols-4 md:gap-12">
          {items.map(s => (
            <div key={s.label} className="flex flex-col items-center text-center">
              <span className="text-2xl font-bold tracking-tight md:text-3xl"><Counter end={s.end} suffix={s.suffix}/></span>
              <span className="text-muted-foreground mt-1.5 text-xs">{s.label}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}