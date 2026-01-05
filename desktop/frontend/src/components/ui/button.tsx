import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap rounded text-sm font-medium transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-sky-500 text-white hover:bg-sky-600 active:bg-sky-700",
        destructive: "bg-red-500 text-white hover:bg-red-600 active:bg-red-700",
        outline: "border-2 border-sky-500 text-sky-600 bg-white hover:bg-sky-50 hover:border-sky-600 active:bg-sky-100",
        secondary: "bg-sky-100 text-sky-700 hover:bg-sky-200 hover:text-sky-800 active:bg-sky-300",
        ghost: "text-slate-600 hover:bg-sky-100 hover:text-sky-700 active:bg-sky-200",
        link: "text-sky-600 underline-offset-4 hover:underline hover:text-sky-700 active:text-sky-800",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-10 rounded-md px-8",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, ...props }, ref) => {
    return (
      <button
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }
