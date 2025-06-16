"use client"

import { useEffect, useState } from "react"
import Link from "next/link"

interface LoginButtonProps {
  callbackUrl?: string
}

export default function LoginButton({callbackUrl}: LoginButtonProps) {
  const [isHovered, setIsHovered] = useState(false)
  const [currentUrl, setCurrentUrl] = useState<string>("");

  // only runs in the browser
  useEffect(() => {
    setCurrentUrl(window.location.href);
  }, []);
  
  const loginUrl = `http://vrpa.ws/client/connect?redirect_url=${encodeURIComponent(callbackUrl || currentUrl)}&service_name=vrpaws-client`

  return (
    <div
      className="flex items-center justify-center min-h-screen bg-gradient-to-br from-cyan-200 via-purple-300 to-pink-300">
      <Link
        href={loginUrl}
        className="relative group"
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        {/* Gradient border container */}
        <div
          className="relative p-[1px] rounded-xl bg-gradient-to-r from-cyan-400 via-purple-500 to-pink-500 transition-all duration-300 group-hover:from-cyan-300 group-hover:via-purple-400 group-hover:to-pink-400 group-hover:shadow-lg group-hover:shadow-purple-500/25">
          {/* Button content */}
          <div
            className="relative px-8 py-4 bg-white rounded-xl transition-all duration-300 group-hover:bg-gradient-to-r group-hover:from-cyan-50 group-hover:via-purple-50 group-hover:to-pink-50">
            <div className="flex items-center space-x-3">
              {/* vrpaws logo/icon */}
              <div
                className="w-6 h-6 rounded-full bg-gradient-to-r from-cyan-500 via-purple-600 to-pink-500 flex items-center justify-center">
                <span className="text-white text-xs font-bold">vr</span>
              </div>

              {/* Button text */}
              <span
                className={`font-semibold text-lg transition-all duration-300 ${
                  isHovered
                    ? "bg-gradient-to-r from-cyan-600 via-purple-700 to-pink-600 bg-clip-text text-transparent"
                    : "text-gray-800"
                }`}
              >
                Sign in with vrpaws
              </span>

              {/* Arrow icon */}
              <svg
                className={`w-5 h-5 transition-all duration-300 ${
                  isHovered ? "text-purple-600 translate-x-1" : "text-gray-600"
                }`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6"/>
              </svg>
            </div>
          </div>
        </div>

        {/* Hover glow effect */}
        <div
          className="absolute inset-0 rounded-xl bg-gradient-to-r from-cyan-400 via-purple-500 to-pink-500 opacity-0 group-hover:opacity-20 transition-opacity duration-300 blur-xl -z-10"/>
      </Link>
    </div>
  )
}
