import { Zap, Globe, Clock, Wand2, Languages, Cog } from "lucide-react"
import { useTranslation } from "react-i18next"
import { AnimateInView } from "@/components/animate-in-view"

export function Features() {
  const { t } = useTranslation()
  const features = [
    { icon: <Zap className="size-5 text-emerald-400"/>, title: t("Unified API"), desc: t("OpenAI / Responses / Claude compatible") },
    { icon: <Cog className="size-5 text-blue-400"/>, title: t("Multi-Model"), desc: t("GPT-5.5 / GPT-4.1 Codex models") },
    { icon: <Globe className="size-5 text-violet-400"/>, title: t("Global Access"), desc: t("Multi-region, smart routing, low latency") },
    { icon: <Wand2 className="size-5 text-amber-400"/>, title: t("AI for All"), desc: t("Same AI capabilities for everyone") },
    { icon: <Languages className="size-5 text-rose-400"/>, title: t("Multilingual"), desc: t("CN/EN/JP/FR and more") },
    { icon: <Clock className="size-5 text-cyan-400"/>, title: t("24/7 Service"), desc: t("7x24 reliable operation") },
  ]
  return (
    <section className="relative z-10 px-6 py-24 md:py-32">
      <div className="mx-auto max-w-6xl">
        <AnimateInView className="mb-16 max-w-lg">
          <p className="text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase">{t("Core Features")}</p>
          <h2 className="text-2xl leading-tight font-bold tracking-tight md:text-3xl">{t("AI Equality for Everyone")}</h2>
        </AnimateInView>
        <div className="grid gap-4 md:grid-cols-3">
          {features.map((f, i) => (
            <AnimateInView key={f.title} delay={i*100} animation="fade-up">
              <div className="border-border/40 bg-card hover:bg-muted/20 group rounded-xl border p-6 transition-colors">
                <div className="text-muted-foreground group-hover:text-foreground mb-4 flex size-10 items-center justify-center rounded-lg bg-gradient-to-br from-emerald-500/10 to-teal-500/10 transition-colors">{f.icon}</div>
                <h3 className="mb-2 text-sm font-semibold">{f.title}</h3>
                <p className="text-muted-foreground text-sm leading-relaxed">{f.desc}</p>
              </div>
            </AnimateInView>
          ))}
        </div>
      </div>
    </section>
  )
}