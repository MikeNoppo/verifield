import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"
import { STATUS_LABEL } from "@/lib/domain/status"
import type { Status } from "@/lib/domain/types"

/** Delapan kunci ditulis literal. Nama kelas hasil interpolasi (`bg-status-${s}`)
    tidak akan ditemukan pemindai Tailwind dan diam-diam hilang dari CSS. */
const statusBadgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center gap-1.5 rounded-4xl border border-transparent font-medium whitespace-nowrap",
  {
    variants: {
      status: {
        requested: "bg-status-requested/12 text-status-requested",
        assigned: "bg-status-assigned/12 text-status-assigned",
        on_the_way: "bg-status-on-the-way/12 text-status-on-the-way",
        on_site: "bg-status-on-site/12 text-status-on-site",
        in_progress: "bg-status-in-progress/12 text-status-in-progress",
        completed: "bg-status-completed/12 text-status-completed",
        failed: "bg-status-failed/14 text-status-failed",
        cancelled: "bg-status-cancelled/12 text-status-cancelled",
      },
      size: {
        sm: "h-5 px-2 text-[11px]",
        md: "h-6 px-2.5 text-xs",
        lg: "h-7 px-3 text-sm",
      },
    },
    defaultVariants: { status: "requested", size: "md" },
  },
)

type Props = {
  status: Status
  size?: VariantProps<typeof statusBadgeVariants>["size"]
  dot?: boolean
  className?: string
}

export function StatusBadge({ status, size = "md", dot = true, className }: Props) {
  return (
    <span className={cn(statusBadgeVariants({ status, size }), className)}>
      {dot ? (
        <span aria-hidden className="size-1.5 shrink-0 rounded-full bg-current" />
      ) : null}
      {STATUS_LABEL[status]}
    </span>
  )
}

export { statusBadgeVariants }
