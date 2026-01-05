import * as React from "react"
import { cn } from "@/lib/utils"

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          "flex min-h-[60px] w-full rounded border-2 border-slate-200 bg-white px-3 py-2 text-sm transition-all duration-200 placeholder:text-slate-400 focus-visible:outline-none focus-visible:border-sky-500 focus-visible:ring-2 focus-visible:ring-sky-500/20 disabled:cursor-not-allowed disabled:opacity-50 hover:border-slate-300 resize-y",
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Textarea.displayName = "Textarea"

export { Textarea }
