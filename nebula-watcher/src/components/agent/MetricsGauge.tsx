import { useMemo } from "react";
import { cn } from "@/lib/utils";

interface MetricsGaugeProps {
  value: number;
  label: string;
  icon?: React.ReactNode;
  size?: "sm" | "md" | "lg";
  showValue?: boolean;
}

export function MetricsGauge({
  value,
  label,
  icon,
  size = "md",
  showValue = true,
}: MetricsGaugeProps) {
  const clampedValue = Math.min(100, Math.max(0, value));
  
  const sizeConfig = {
    sm: { svg: 80, stroke: 6, text: "text-lg", label: "text-xs" },
    md: { svg: 120, stroke: 8, text: "text-2xl", label: "text-sm" },
    lg: { svg: 160, stroke: 10, text: "text-3xl", label: "text-base" },
  };

  const config = sizeConfig[size];
  const radius = (config.svg - config.stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (clampedValue / 100) * circumference;

  const getColor = useMemo(() => {
    if (clampedValue >= 90) return { stroke: "stroke-destructive", text: "text-destructive", glow: "hsl(var(--destructive))" };
    if (clampedValue >= 70) return { stroke: "stroke-warning", text: "text-warning", glow: "hsl(var(--warning))" };
    return { stroke: "stroke-success", text: "text-success", glow: "hsl(var(--success))" };
  }, [clampedValue]);

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative" style={{ width: config.svg, height: config.svg }}>
        <svg
          className="transform -rotate-90"
          width={config.svg}
          height={config.svg}
        >
          {/* Background ring */}
          <circle
            cx={config.svg / 2}
            cy={config.svg / 2}
            r={radius}
            stroke="currentColor"
            strokeWidth={config.stroke}
            fill="none"
            className="text-muted/30"
          />
          {/* Value ring */}
          <circle
            cx={config.svg / 2}
            cy={config.svg / 2}
            r={radius}
            strokeWidth={config.stroke}
            fill="none"
            strokeLinecap="round"
            className={cn("gauge-ring transition-all duration-500", getColor.stroke)}
            style={{
              strokeDasharray: circumference,
              strokeDashoffset,
              filter: `drop-shadow(0 0 8px ${getColor.glow})`,
            }}
          />
        </svg>
        
        {/* Center content */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          {icon && <div className="mb-1">{icon}</div>}
          {showValue && (
            <span className={cn("font-mono font-bold", config.text, getColor.text)}>
              {clampedValue.toFixed(0)}%
            </span>
          )}
        </div>
      </div>
      
      <span className={cn("font-medium text-muted-foreground", config.label)}>
        {label}
      </span>
    </div>
  );
}
