import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useMemo,
  useReducer,
  useRef,
} from "react";
import { useApiClient } from "@/features/calculator/context/ApiClientContext";
import {
  binaryExpression,
  completedExpression,
  pendingExpression,
  unaryExpression,
} from "@/features/calculator/state/formatting";
import { calculatorReducer, createInitialState } from "@/features/calculator/state/reducer";
import type {
  CalculatorAction,
  CalculatorError,
  CalculatorState,
  KeyAction,
} from "@/features/calculator/state/types";
import { ApiError, NETWORK_ERROR } from "@/lib/api/client";
import type { BinaryOperationName, CalculateRequest } from "@/lib/api/types";
import { createIdFactory } from "@/lib/ids";

export interface CalculatorContextValue {
  state: CalculatorState;
  /** The single entry point for user intent — clicks and keystrokes alike. */
  press: (action: KeyAction) => void;
}

const CalculatorContext = createContext<CalculatorContextValue | null>(null);

/** Keys for history rows. Never throws, so it cannot be mistaken for an API failure. */
const nextHistoryId = createIdFactory("history");

/** Where the chain stands once a result comes back. */
interface ChainState {
  accumulator: number | null;
  pendingOperation: BinaryOperationName | null;
  expression: string | null;
  /** Whether the result is itself the right operand of `pendingOperation`. */
  hasRightOperand: boolean;
}

/** One round trip: what to send, how to label it, what to do with the answer. */
interface CalculationPlan {
  request: CalculateRequest;
  /** Left-hand side, e.g. `2 + 3`; becomes the history expression. */
  expression: string;
  next: (result: number) => ChainState;
}

function toCalculatorError(cause: unknown): CalculatorError {
  if (cause instanceof ApiError) {
    return { code: cause.code, message: cause.message };
  }
  return {
    code: NETWORK_ERROR,
    message: cause instanceof Error ? cause.message : "The calculation could not be performed",
  };
}

export interface CalculatorProviderProps {
  children: ReactNode;
}

/**
 * Owns the calculator state machine and the async actions that talk to the API.
 *
 * `Number(input)` (parsing) and `String(result)` (rendering) are the only
 * conversions performed here; every arithmetic answer comes from the server.
 */
export function CalculatorProvider({ children }: CalculatorProviderProps) {
  const client = useApiClient();
  const [state, dispatch] = useReducer(calculatorReducer, undefined, createInitialState);

  // Actions run outside the render that produced the state they need — a
  // keystroke, or a response arriving later. They read this ref, which is
  // advanced by the same pure reducer React uses, so it is correct even when
  // several keys are pressed before React has re-rendered.
  const stateRef = useRef(state);
  const apply = useCallback((action: CalculatorAction) => {
    stateRef.current = calculatorReducer(stateRef.current, action);
    dispatch(action);
  }, []);

  const runCalculation = useCallback(
    async (plan: CalculationPlan) => {
      apply({ type: "CALCULATION_STARTED", expression: completedExpression(plan.expression) });
      try {
        const response = await client.calculate(plan.request);
        const chain = plan.next(response.result);
        apply({
          type: "CALCULATION_SUCCEEDED",
          result: response.result,
          entry: {
            id: nextHistoryId(),
            expression: plan.expression,
            result: response.result,
          },
          accumulator: chain.accumulator,
          pendingOperation: chain.pendingOperation,
          expression: chain.expression,
          hasRightOperand: chain.hasRightOperand,
        });
      } catch (cause) {
        apply({ type: "CALCULATION_FAILED", error: toCalculatorError(cause) });
      }
    },
    [client, apply],
  );

  const press = useCallback(
    (action: KeyAction) => {
      const current = stateRef.current;
      if (current.status === "calculating") {
        return;
      }

      switch (action.type) {
        case "digit":
          apply({ type: "DIGIT", digit: action.digit });
          return;

        case "decimal":
          apply({ type: "DECIMAL" });
          return;

        case "backspace":
          apply({ type: "BACKSPACE" });
          return;

        case "clear":
          apply({ type: "CLEAR" });
          return;

        case "operator": {
          if (current.status === "error") {
            return;
          }
          const operand = Number(current.input);
          const { accumulator, pendingOperation } = current;
          // A right operand has been typed: settle the pending operation first,
          // then continue the chain with the operator just pressed.
          if (pendingOperation !== null && accumulator !== null && current.hasRightOperand) {
            void runCalculation({
              request: { operation: pendingOperation, operands: [accumulator, operand] },
              expression: binaryExpression(accumulator, pendingOperation, operand),
              // The answer becomes the left operand of the operator just
              // pressed, which is now waiting for a right operand of its own.
              next: (result) => ({
                accumulator: result,
                pendingOperation: action.operation,
                expression: pendingExpression(result, action.operation),
                hasRightOperand: false,
              }),
            });
            return;
          }
          apply({
            type: "OPERATOR_SELECTED",
            operation: action.operation,
            accumulator: operand,
            expression: pendingExpression(operand, action.operation),
          });
          return;
        }

        case "unary": {
          if (current.status === "error") {
            return;
          }
          const operand = Number(current.input);
          const expression = unaryExpression(action.operation, operand);
          void runCalculation({
            request: { operation: action.operation, operands: [operand] },
            expression,
            // A unary operation rewrites the current input; a binary operation
            // waiting for that input keeps waiting.
            next: () => ({
              accumulator: current.accumulator,
              pendingOperation: current.pendingOperation,
              expression:
                current.pendingOperation === null
                  ? completedExpression(expression)
                  : current.expression,
              hasRightOperand: current.pendingOperation !== null,
            }),
          });
          return;
        }

        case "equals": {
          if (current.status === "error") {
            return;
          }
          const { accumulator, pendingOperation } = current;
          if (pendingOperation === null || accumulator === null) {
            return;
          }
          const operand = Number(current.input);
          const expression = binaryExpression(accumulator, pendingOperation, operand);
          void runCalculation({
            request: { operation: pendingOperation, operands: [accumulator, operand] },
            expression,
            next: () => ({
              accumulator: null,
              pendingOperation: null,
              expression: completedExpression(expression),
              hasRightOperand: false,
            }),
          });
          return;
        }
      }
    },
    [runCalculation, apply],
  );

  const value = useMemo<CalculatorContextValue>(() => ({ state, press }), [state, press]);

  return <CalculatorContext value={value}>{children}</CalculatorContext>;
}

export function useCalculator(): CalculatorContextValue {
  const value = useContext(CalculatorContext);
  if (value === null) {
    throw new Error("useCalculator must be used inside <Calculator>");
  }
  return value;
}
