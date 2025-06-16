"use client"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Home, Camera, Settings } from "lucide-react"
import Link from "next/link"
import type { LoginSuccessProps } from "./types/login-success"
import { DogPawIcon } from "./components/dog-paw-icon"

export default function Component({
  title = "Login Successful!",
  description = "Welcome back! You have been successfully logged in to your account.",
  subtitleLabel = "You can now access all features of your account.",
  streamUrl = "https://vrpa.ws/stream",
  photosUrl = "https://vrpa.ws/my-photos",
  settingsUrl = "https://vrpa.ws/settings",
  supportUrl = "/support",
  streamLabel = "Stream",
  photosLabel = "My Photos",
  settingsLabel = "Account Settings",
  supportText = "Contact Support",
}: LoginSuccessProps = {}) {
  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-purple-600 via-blue-500 to-teal-400">
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
          background: white;
          border-radius: 0.375rem;
          margin: 1px;
        }
      `}</style>

      <Card className="w-full max-w-md backdrop-blur-sm bg-white/90 border-0 shadow-2xl">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-r from-blue-400 to-emerald-500 shadow-lg">
            <DogPawIcon className="h-8 w-8 text-white" />
          </div>
          <CardTitle className="text-2xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent">
            {title}
          </CardTitle>
          <CardDescription className="text-base text-gray-600">{description}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-center text-sm text-gray-500">{subtitleLabel}</div>

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
              <div className="gradient-border-content">
                <Button
                  variant="ghost"
                  asChild
                  className="w-full border-0 bg-transparent hover:bg-gray-50 text-gray-800 hover:text-gray-900 font-medium transition-all duration-200"
                >
                  <Link href={photosUrl} className="flex items-center justify-center">
                    <Camera className="absolute left-4 h-4 w-4 text-blue-600" />
                    <span className="text-center">{photosLabel}</span>
                  </Link>
                </Button>
              </div>
            </div>

            <div className="gradient-border-2 hover:scale-105 transition-all duration-200">
              <div className="gradient-border-content">
                <Button
                  variant="ghost"
                  asChild
                  className="w-full border-0 bg-transparent hover:bg-gray-50 text-gray-800 hover:text-gray-900 font-medium transition-all duration-200"
                >
                  <Link href={settingsUrl} className="flex items-center justify-center">
                    <Settings className="absolute left-4 h-4 w-4 text-teal-600" />
                    <span className="text-center">{settingsLabel}</span>
                  </Link>
                </Button>
              </div>
            </div>
          </div>

          <div className="pt-4 border-t border-gray-200 text-center">
            <p className="text-xs text-gray-500">
              Need help?{" "}
              <Link
                href={supportUrl}
                className="text-purple-600 hover:text-purple-700 font-medium transition-colors duration-200 hover:underline"
              >
                {supportText}
              </Link>
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
