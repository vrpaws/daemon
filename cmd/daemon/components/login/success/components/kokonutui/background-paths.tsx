"use client"

import { motion } from "framer-motion"
import { Button } from "@/components/ui/button"
import React, { useMemo } from "react"

function getRandomGradient(i: number) {
  const gradients = [
    ["#00E2FD", "#6D90FA", "#FF22EE", "#FF8D7A", "#FFC851"],
    ["#2A7B9B", "#57C785", "#EDDD53"],
    ["#C5F9D7", "#F7D486", "#F27A7D"],
    ["#C2E59C", "#64B3F4"],
    ["#CAEFD7", "#F5BFD7", "#ABC9E9"],
  ];
  const g = gradients[i % gradients.length];
  return `url(#gradient${i})`;
}

function FloatingPaths({ position }: { position: number }) {
  const gradients = [
    ["#00E2FD", "#6D90FA", "#FF22EE", "#FF8D7A", "#FFC851"],
    ["#2A7B9B", "#57C785", "#EDDD53"],
    ["#C5F9D7", "#F7D486", "#F27A7D"],
    ["#C2E59C", "#64B3F4"],
    ["#CAEFD7", "#F5BFD7", "#ABC9E9"],
  ];
  const paths = useMemo(() => Array.from({ length: 36 }, (_, i) => ({
    id: i,
    d: `M-${380 - i * 5 * position} -${189 + i * 6}C-${380 - i * 5 * position} -${189 + i * 6} -${312 - i * 5 * position} ${216 - i * 6} ${152 - i * 5 * position} ${343 - i * 6}C${616 - i * 5 * position} ${470 - i * 6} ${684 - i * 5 * position} ${875 - i * 6} ${684 - i * 5 * position} ${875 - i * 6}`,
    gradient: gradients[i % gradients.length],
    width: 0.5 + i * 0.03,
  })), [position]);

  return (
    <div className="absolute inset-0 pointer-events-none">
      <svg className="w-full h-full" viewBox="0 0 696 316" fill="none">
        <defs>
          {paths.map((path, i) => (
            <linearGradient id={`gradient${i}`} key={i} x1="0" y1="0" x2="1" y2="1">
              {path.gradient.map((color: string, idx: number) => (
                <stop key={idx} offset={`${(idx / (path.gradient.length - 1)) * 100}%`} stopColor={color} />
              ))}
            </linearGradient>
          ))}
        </defs>
        <title>Background Paths</title>
        {paths.map((path, i) => (
          <motion.path
            key={path.id}
            d={path.d}
            stroke={`url(#gradient${i})`}
            strokeWidth={path.width}
            strokeOpacity={0.1 + path.id * 0.03}
            initial={{ pathLength: 0.3, opacity: 0.6 }}
            animate={{
              pathLength: 1,
              opacity: [0.3, 0.6, 0.3],
              pathOffset: [0, 1, 0],
            }}
            transition={{
              duration: 20 + Math.random() * 10,
              repeat: Number.POSITIVE_INFINITY,
              ease: "linear",
            }}
          />
        ))}
      </svg>
    </div>
  );
}

export function BackgroundAnimation() {
  return (
    <div className="absolute inset-0">
      <FloatingPaths position={1} />
      <FloatingPaths position={-1} />
    </div>
  );
}

export function AnimatedTitle({ title = "VRPaws" }: { title?: string }) {
  const words = title.split(" ");

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 2 }}
      className="max-w-4xl mx-auto"
    >
      <h1 className="text-5xl sm:text-7xl md:text-8xl font-bold mb-8 tracking-tighter">
        {words.map((word, wordIndex) => (
          <span key={wordIndex} className="inline-block mr-4 last:mr-0">
            {word.split("").map((letter, letterIndex) => (
              <motion.span
                key={`${wordIndex}-${letterIndex}`}
                initial={{ y: 100, opacity: 0 }}
                animate={{ y: 0, opacity: 1 }}
                transition={{
                  delay: wordIndex * 0.1 + letterIndex * 0.03,
                  type: "spring",
                  stiffness: 150,
                  damping: 25,
                }}
                className="inline-block text-transparent bg-clip-text 
                                    bg-gradient-to-r from-white to-white/80 
                                    dark:from-white dark:to-white/80"
              >
                {letter}
              </motion.span>
            ))}
          </span>
        ))}
      </h1>
    </motion.div>
  );
}

export function StreamButton({ buttonTitle = "Stream" }: { buttonTitle?: string }) {
  return (
    <div
      className="inline-block group relative bg-gradient-to-b from-white/10 to-black/10 p-px rounded-2xl backdrop-blur-lg 
                  overflow-hidden shadow-lg hover:shadow-xl transition-shadow duration-300 cursor-pointer"
    >
      <Button
        variant="ghost"
        className="rounded-[1.15rem] px-8 py-6 text-lg font-semibold backdrop-blur-md 
                    bg-transparent hover:bg-white/10 text-white transition-all duration-300 
                    group-hover:-translate-y-0.5 border border-white/10
                    hover:shadow-md dark:hover:shadow-neutral-800/50 cursor-pointer"
      >
        <span className="opacity-90 group-hover:opacity-100 transition-opacity">{buttonTitle}</span>
        <span
          className="ml-3 opacity-70 group-hover:opacity-100 group-hover:translate-x-1.5 
                          transition-all duration-300"
        >
          →
        </span>
      </Button>
    </div>
  );
}
