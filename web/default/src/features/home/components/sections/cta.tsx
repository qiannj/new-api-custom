import { Link } from "@tanstack/react-router"
import { ArrowRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"

interface CTAProps { isAuthenticated?: boolean }

export function CTA(props: CTAProps) {
  const {t} = useTranslation()
  return (
    <section className="relative z-10 overflow-hidden px-6 py-24 md:py-32">
      <div aria-hidden className="pointer-events-none absolute inset-0 -z-10 opacity-[0.03]"
        style={{background:"radial-gradient(ellipse 80% 50% at 50% 50%, oklch(0.7 0.2 160) 0%, transparent 70%)"}}/>
      <div className="mx-auto flex max-w-2xl flex-col items-center text-center">
        <h2 className="text-2xl leading-tight font-bold tracking-tight md:text-3xl">{t("AI Equality, Start Now")}</h2>
        <p className="text-muted-foreground/80 mt-4 max-w-md text-sm leading-relaxed">{t("Join minyingAPI, get the same AI power as the world")}</p>
        <div className="mt-8 flex items-center gap-3">
          {props.isAuthenticated ? (
            <Button className="group rounded-lg" render={<Link to="/console"/>}>{t("Dashboard")}<ArrowRight className="ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5"/></Button>
          ) : (
            <>
              <Button className="group rounded-lg bg-gradient-to-r from-emerald-500 to-teal-500 hover:from-emerald-600 hover:to-teal-600" render={<Link to="/sign-up"/>}>{t("Get Started")}<ArrowRight className="ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5"/></Button>
              <Button variant="outline" className="border-border/50 hover:border-border hover:bg-muted/50 rounded-lg" render={<Link to="/pricing"/>}>{t("View Models")}</Button>
            </>
          )}
        </div>
      </div>
    </section>
  )
}