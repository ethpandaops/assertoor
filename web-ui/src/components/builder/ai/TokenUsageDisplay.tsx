import React, { useEffect } from 'react';
import { useAIStore } from '../../../stores/aiStore';

// Format large numbers with commas
function formatNumber(num: number): string {
  return num.toLocaleString();
}

export const TokenUsageDisplay: React.FC = () => {
  const { usageLastDay, usageLastMonth, loadUsage } = useAIStore();

  useEffect(() => {
    loadUsage();
  }, [loadUsage]);

  return (
    <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
      <div className="flex items-center gap-1">
        <span className="font-medium">24h:</span>
        <span>{formatNumber(usageLastDay?.totalTokens || 0)} tokens</span>
      </div>
      <div className="flex items-center gap-1">
        <span className="font-medium">30d:</span>
        <span>{formatNumber(usageLastMonth?.totalTokens || 0)} tokens</span>
      </div>
    </div>
  );
};
