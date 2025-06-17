"use client"

import { motion, Variants } from "framer-motion"
import Image from "next/image"

interface HeroBadgeProps {
  text: string
  logoSrc?: string
  logoAlt?: string
  delay?: number
}

export function HeroBadge({
  text,
  logoSrc = "https://kokonutui.com/logo.svg",
  logoAlt = "Logo",
  delay = 0,
}: HeroBadgeProps) {
  const fadeUpVariants: Variants = {
    hidden: { opacity: 0, y: 30 },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 1,
        delay: 0.5 + delay,
        ease: [0.25, 0.4, 0.25, 1],
      },
    },
  }

  return (
    <motion.div
      variants={fadeUpVariants}
      initial="hidden"
      animate="visible"
      className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/[0.03] border border-white/[0.08] mb-8 md:mb-12"
    >
      <Image src={logoSrc || "/placeholder.svg"} alt={logoAlt} width={20} height={20} />
      <span className="text-sm text-white/60 tracking-wide">{text}</span>
    </motion.div>
  )
}
