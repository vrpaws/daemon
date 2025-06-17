"use client"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Camera, Home, Settings, LogIn } from "lucide-react"
import Link from "next/link"
import type { LoginSuccessProps } from "./types/login-success"
import { HeroBackground } from "./components/kokonutui/hero-background"
import { FloatingShapes } from "./components/kokonutui/floating-shapes"
import { HeroTitle } from "./components/kokonutui/hero-title"
import { HeroDescription } from "./components/kokonutui/hero-description"
import { useSearchParams } from "next/navigation"
import { useEffect, useState } from "react"

export default function Component({
                                    title = "VRPaws",
                                    description = "Connect, share, and relive your VRChat moments with friends.",
                                    subtitleLabel = "You can now close this window.",
                                    photosUrl = "https://vrpa.ws/my-photos",
                                    settingsUrl = "https://vrpa.ws/settings",
                                    supportUrl = "/support",
                                    photosLabel = "My Photos",
                                    settingsLabel = "Account Settings",
                                    supportText = "Contact Support",
                                    streamUrl = "https://vrpa.ws/stream",
                                    streamLabel = "Stream",
                                  }: LoginSuccessProps = {}) {
  const searchParams = useSearchParams()
  const hasAccessToken = searchParams.get("access_token")
  const [currentUrl, setCurrentUrl] = useState("")
  const [loginUrl, setLoginUrl] = useState("")

  useEffect(() => {
    setCurrentUrl(window.location.href)
    setLoginUrl(`https://vrpa.ws/client/connect?redirect_url=${encodeURIComponent(currentUrl)}&service_name=vrpaws-client`)
  }, [])

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden bg-[#030303]">
      <style jsx>{`
        @keyframes gradient-flow {
          0% {
            background-position: 0% 50%;
          }
          50% {
            background-position: 100% 50%;
          }
          100% {
            background-position: 0% 50%;
          }
        }
        
        .gradient-border-1 {
          position: relative;
          background: white;
          border-radius: 0.5rem;
          overflow: hidden;
        }
        
        .gradient-border-1::before {
          content: '';
          position: absolute;
          inset: -1px;
          background: linear-gradient(45deg, #3b82f6, #14b8a6, #8b5cf6, #3b82f6, #14b8a6);
          background-size: 300% 300%;
          border-radius: inherit;
          animation: gradient-flow 3s ease-in-out infinite;
          z-index: 0;
        }
        
        .gradient-border-2 {
          position: relative;
          background: white;
          border-radius: 0.5rem;
          overflow: hidden;
        }
        
        .gradient-border-2::before {
          content: '';
          position: absolute;
          inset: -1px;
          background: linear-gradient(-45deg, #14b8a6, #8b5cf6, #3b82f6, #14b8a6, #8b5cf6);
          background-size: 300% 300%;
          border-radius: inherit;
          animation: gradient-flow 3s ease-in-out infinite reverse;
          z-index: 0;
        }
        
        .gradient-border-content {
          position: relative;
          z-index: 1;
          border-radius: 0.375rem;
          margin: 1px;
        }
      `}</style>
      <HeroBackground />
      <FloatingShapes />

      <div className="relative z-10 w-full flex flex-col items-center">
        <div className="max-w-3xl z-10 mx-auto text-center translate-y-4">
          <HeroTitle title1="Welcome to" title2={title} />
        </div>

        <Card className="w-full sm:w-[420px] md:w-[480px] lg:w-[520px] xl:w-[600px] backdrop-blur-md bg-white/10 border border-white/20 shadow-2xl">
          <CardContent className="space-y-4">
            <CardHeader className="text-center">
              <HeroDescription
                text={description}
                className="text-base sm:text-lg md:text-xl text-white/80 leading-relaxed font-light tracking-wide max-w-xl mx-auto px-4"
              />
            </CardHeader>

            {hasAccessToken ? (
              <>
                <div className="text-center">
                  <HeroDescription text={subtitleLabel} delay={0.6} />
                </div>

                <div className="grid gap-3">
                  <Button
                    asChild
                    className="w-full bg-gradient-to-r from-purple-500 to-blue-500 hover:from-purple-600 hover:to-blue-600 text-white border-0 shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200"
                  >
                    <Link href={streamUrl} className="flex items-center justify-center relative">
                      <Home className="absolute left-4 h-4 w-4" />
                      <span className="text-center">{streamLabel}</span>
                    </Link>
                  </Button>

                  <div className="gradient-border-1 hover:scale-105 transition-all duration-200">
                    <div className="gradient-border-content backdrop-blur-sm">
                      <Button
                        variant="ghost"
                        asChild
                        className="w-full border-0 bg-black hover:bg-black text-white hover:text-white font-medium transition-all duration-200"
                      >
                        <Link href={photosUrl} className="flex items-center justify-center">
                          <Camera className="absolute left-4 h-4 w-4 text-blue-300" />
                          <span className="text-center">{photosLabel}</span>
                        </Link>
                      </Button>
                    </div>
                  </div>

                  <div className="gradient-border-2 hover:scale-105 transition-all duration-200">
                    <div className="gradient-border-content backdrop-blur-sm">
                      <Button
                        variant="ghost"
                        asChild
                        className="w-full border-0 bg-black hover:bg-black text-white hover:text-white font-medium transition-all duration-200"
                      >
                        <Link href={settingsUrl} className="flex items-center justify-center">
                          <Settings className="absolute left-4 h-4 w-4 text-teal-300" />
                          <span className="text-center">{settingsLabel}</span>
                        </Link>
                      </Button>
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <div className="grid gap-3">
                <Button
                  asChild
                  className="w-full bg-gradient-to-r from-purple-500 to-blue-500 hover:from-purple-600 hover:to-blue-600 text-white border-0 shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200"
                >
                  <Link href={loginUrl} className="flex items-center justify-center relative">
                    <LogIn className="absolute left-4 h-4 w-4" />
                    <span className="text-center">Log in</span>
                  </Link>
                </Button>
              </div>
            )}

            <div className="pt-4 border-t border-white/20 text-center">
              <p className="text-xs text-white/70 drop-shadow">
                Need help?{" "}
                <Link
                  href={supportUrl}
                  className="text-white hover:text-white/80 font-medium transition-colors duration-200 hover:underline"
                >
                  {supportText}
                </Link>
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
