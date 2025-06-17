"use client"

import { HeroBackground } from "./hero-background"
import { FloatingShapes } from "./floating-shapes"
import { HeroBadge } from "./hero-badge"
import { HeroTitle } from "./hero-title"
import { HeroDescription } from "./hero-description"

interface HeroGeometricProps {
  badge?: string
  title1?: string
  title2?: string
  description?: string
  logoSrc?: string
  logoAlt?: string
}

export default function HeroGeometric({
  badge = "Kokonut UI",
  title1 = "Elevate Your",
  title2 = "Digital Vision",
  description = "Crafting exceptional digital experiences through innovative design and cutting-edge technology.",
  logoSrc = "https://kokonutui.com/logo.svg",
  logoAlt = "Kokonut UI",
}: HeroGeometricProps) {
  return (
    <div className="relative min-h-screen w-full flex items-center justify-center overflow-hidden bg-[#030303]">
      <HeroBackground />
      <FloatingShapes />

      <div className="relative z-10 container mx-auto px-4 md:px-6">
        <div className="max-w-3xl mx-auto text-center">
          <HeroBadge text={badge} logoSrc={logoSrc} logoAlt={logoAlt} delay={0} />

          <HeroTitle title1={title1} title2={title2} delay={0.2} />

          <HeroDescription text={description} delay={0.4} />
        </div>
      </div>
    </div>
  )
}
