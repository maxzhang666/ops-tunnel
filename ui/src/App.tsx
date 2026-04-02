import { useEffect, useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

function App() {
  const [status, setStatus] = useState<"checking" | "connected" | "disconnected">("checking")

  useEffect(() => {
    fetch("/healthz")
      .then((res) => {
        if (res.ok) setStatus("connected")
        else setStatus("disconnected")
      })
      .catch(() => setStatus("disconnected"))
  }, [])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <Card className="w-80">
        <CardHeader>
          <CardTitle className="text-center text-2xl">OpsTunnel</CardTitle>
        </CardHeader>
        <CardContent className="flex justify-center">
          {status === "checking" && <Badge variant="outline">Checking...</Badge>}
          {status === "connected" && <Badge variant="default">Connected</Badge>}
          {status === "disconnected" && <Badge variant="destructive">Disconnected</Badge>}
        </CardContent>
      </Card>
    </div>
  )
}

export default App
