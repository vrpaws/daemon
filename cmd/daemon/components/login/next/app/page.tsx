import { Suspense } from "react"
import Component from "../login-success"

function LoginSuccessPage() {
  return (
    <Component
      description="Connect, share, and relive your VRChat moments with friends."
      subtitleLabel="You can now close this window."
      streamLabel="Stream"
      photosLabel="My Photos"
      settingsLabel="Settings"
      supportText="Contact Support"
    />
  )
}

export default function Page() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <LoginSuccessPage />
    </Suspense>
  )
}
