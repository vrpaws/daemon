"use client"

import { Easing, motion, Variants } from "framer-motion"
import { Pacifico } from "next/font/google"
import { cn } from "@/lib/utils"

const pacifico = Pacifico({
  subsets: ["latin"],
  weight: ["400"],
  variable: "--font-pacifico",
})

interface HeroTitleProps {
  title1: string
  title2: string
  delay?: number
}

export function HeroTitle({ title1, title2, delay = 0.2 }: HeroTitleProps) {
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
    <motion.div variants={fadeUpVariants} initial="hidden" animate="visible">
      <h1 className="text-4xl sm:text-6xl md:text-8xl font-bold mb-6 md:mb-8">
        <span className="bg-clip-text text-transparent bg-gradient-to-b from-white to-white/80 tracking-tight">{title1}</span>
        <br />
        <span
          className={cn(
            "bg-clip-text text-transparent bg-gradient-to-r from-indigo-300 via-white/90 to-rose-300",
            pacifico.className,
          )}
        >
          {title2.toLowerCase()}
        </span>
      </h1>
    </motion.div>
  )
}
