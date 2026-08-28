"use client"

import type { Route } from "next"

import { useActor } from "@/components/verifield/actor-provider"
import { withActor } from "@/lib/actor/link"

export function useActorHref(): (href: string) => Route {
  const actorId = useActor().id
  return (href) => withActor(href, actorId)
}
