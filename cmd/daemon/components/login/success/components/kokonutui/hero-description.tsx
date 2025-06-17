"use client"

import { motion, Variants } from "framer-motion"

interface HeroDescriptionProps {
  text: string
  delay?: number
  className?: string
}

export function HeroDescription({
                                  text,
                                  delay = 0.4,
                                  className = "text-base sm:text-lg md:text-xl text-white/40 leading-relaxed font-light tracking-wide max-w-xl mx-auto px-4",
                                }: HeroDescriptionProps) {
  const fadeUpVariants: Variants = {
    hidden: {opacity: 0, y: 30},
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
    <motion.div variants={fadeUpVariants} initial="hidden" animate="visible">
      <p className={className}>{text}</p>
    </motion.div>
  )
}
