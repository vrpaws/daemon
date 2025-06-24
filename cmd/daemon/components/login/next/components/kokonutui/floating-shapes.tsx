"use client"

import { FloatingShape } from "./floating-shape"

interface FloatingShapesProps {
  className?: string
}

export function FloatingShapes({ className }: FloatingShapesProps) {
  const shapes = [
    {
      delay: 0.3,
      width: 600,
      height: 140,
      rotate: 12,
      gradient: "from-indigo-500/[0.15]",
      className: "left-[-10%] md:left-[-5%] top-[15%] md:top-[20%]",
    },
    {
      delay: 0.5,
      width: 500,
      height: 120,
      rotate: -15,
      gradient: "from-rose-500/[0.15]",
      className: "right-[-5%] md:right-[0%] top-[70%] md:top-[75%]",
    },
    {
      delay: 0.4,
      width: 300,
      height: 80,
      rotate: -8,
      gradient: "from-violet-500/[0.15]",
      className: "left-[5%] md:left-[10%] bottom-[5%] md:bottom-[10%]",
    },
    {
      delay: 0.6,
      width: 200,
      height: 60,
      rotate: 20,
      gradient: "from-amber-500/[0.15]",
      className: "right-[15%] md:right-[20%] top-[10%] md:top-[15%]",
    },
    {
      delay: 0.7,
      width: 150,
      height: 40,
      rotate: -25,
      gradient: "from-cyan-500/[0.15]",
      className: "left-[20%] md:left-[25%] top-[5%] md:top-[10%]",
    },
  ]

  return (
    <div className="absolute inset-0 overflow-hidden">
      {shapes.map((shape, index) => (
        <FloatingShape key={index} {...shape} />
      ))}
    </div>
  )
}
