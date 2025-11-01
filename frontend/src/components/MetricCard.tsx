import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { LucideIcon } from "lucide-react";

interface MetricCardProps {
  title: string;
  value: string | number;
  icon: LucideIcon;
  trend?: 'up' | 'down' | 'neutral';
  className?: string;
}

export const MetricCard = ({ title, value, icon: Icon, trend, className }: MetricCardProps) => {
  const trendColors = {
    up: 'text-success',
    down: 'text-destructive',
    neutral: 'text-muted-foreground',
  };

  return (
    <Card className={cn(
      "relative p-6 glass-effect border-primary/20 hover:border-primary/60 transition-all duration-500 group overflow-hidden",
      "hover:card-glow-strong hover:scale-[1.02]",
      className
    )}>
      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-secondary/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
      
      <div className="relative flex items-start justify-between">
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground uppercase tracking-wider">{title}</p>
          <p className={cn(
            "text-4xl font-bold font-mono transition-all duration-300",
            "group-hover:text-glow",
            trend && trendColors[trend]
          )}>
            {value}
          </p>
        </div>
        <div className="relative p-3 bg-primary/10 rounded-lg border border-primary/30 group-hover:bg-primary/20 transition-all duration-300 animate-float">
          <Icon className="w-6 h-6 text-primary group-hover:scale-110 transition-transform duration-300" />
          <div className="absolute inset-0 bg-primary/20 rounded-lg blur-xl opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
        </div>
      </div>
      
      {/* Corner accents */}
      <div className="absolute top-0 left-0 w-8 h-8 border-t-2 border-l-2 border-primary/30 group-hover:border-primary/60 transition-colors duration-300" />
      <div className="absolute bottom-0 right-0 w-8 h-8 border-b-2 border-r-2 border-primary/30 group-hover:border-primary/60 transition-colors duration-300" />
    </Card>
  );
};
