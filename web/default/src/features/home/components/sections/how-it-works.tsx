import { useTranslation } from "react-i18next"
import { AnimateInView } from "@/components/animate-in-view"

export function HowItWorks() {
  const {t} = useTranslation()
  const steps = [
    {num:"01", title:t("Register"), desc:t("Sign up at chat.minying.cc")},
    {num:"02", title:t("Get API Key"), desc:t("Create a token in console")},
    {num:"03", title:t("Configure"), desc:t("Setup Hermes / OpenClaw / SDK")},
    {num:"04", title:t("Start Using"), desc:t("Send API requests and enjoy!")},
  ]
  return (
    <section className="relative z-10 px-6 py-24 md:py-32">
      <div className="mx-auto max-w-6xl">
        <AnimateInView className="mb-16 text-center">
          <p className="text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase">{t("Quick Start")}</p>
          <h2 className="text-2xl leading-tight font-bold tracking-tight md:text-3xl">{t("Get Started in 4 Steps")}</h2>
        </AnimateInView>
        <div className="relative grid gap-8 md:grid-cols-4">
          {steps.map((s,i) => (
            <AnimateInView key={s.num} delay={i*100} animation="fade-up" className="relative flex flex-col items-center text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-gradient-to-br from-emerald-500 to-teal-500 text-sm font-bold text-white shadow-lg">{s.num}</div>
              <h3 className="mt-4 mb-2 text-sm font-semibold">{s.title}</h3>
              <p className="text-muted-foreground max-w-[200px] text-xs leading-relaxed">{s.desc}</p>
            </AnimateInView>
          ))}
        </div>
      </div>
    </section>
  )
}