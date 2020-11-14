import numpy as np
import matplotlib.pyplot as plt


def generate_dots(n = -1):
    dots = [(0, 0)]
    count = np.random.randint(3, 20)
    if n > 1:
        count = n - 1
    for _ in range(count):
        x = np.random.randint(-10, 11)
        y = np.random.randint(-10, 11)
        dots.append((x, y))
    return np.array(dots)


dots = generate_dots(6)
x = dots[:, 0]
y = dots[:, 1]


def calculate_parameters(x, y):
    sum = 0
    n = len(x)
    for i in range(n):
        for j in range(i + 1, n):
            sum += np.sqrt((x[j] - x[i]) ** 2 + (y[j] - y[i]) ** 2)
    edge_count = n * (n - 1) / 2
    average_length = sum / edge_count
    return average_length, sum


def loss(x, y, average, s):
    sum = 0
    n = len(x)
    _, length = calculate_parameters(x, y)
    for i in range(n):
        for j in range(i + 1, n):
            dist = ((x[j] - x[i]) ** 2 + (y[j] - y[i]) ** 2) ** 0.5
            sum += (dist - average) ** 2
    return sum + abs(length - s)


def gradient(x, y, average, s):
    g = []
    h = 1e-12
    f0 = loss(x, y, average, s)
    for i in range(len(x)):
        xx = np.array(x, copy=True)
        yy = np.array(y, copy=True)
        xx[i] += h
        yy[i] += h
        dx = (loss(xx, y, average, s) - f0) / h
        dy = (loss(x, yy, average, s) - f0) / h
        g.append((dx, dy))
    g = np.array(g).T
    return g / np.linalg.norm(g)


def gradient_descent(x, y, s):
    iteration = 0
    precision = 1e10
    log = []
    average_length, _ = calculate_parameters(x, y)

    alpha = 0.2
    while precision > 1e-6:
        iteration += 1
        prev_loss = loss(x, y, average_length, s)

        grad = gradient(x, y, average_length, s)
        grad[:, 0] = 0  # point (0,0) is fixed in place

        log.append((iteration, prev_loss))
        x = x - alpha * grad[0]
        y = y - alpha * grad[1]

        current_loss = loss(x, y, average_length, s)
        precision = np.abs(current_loss - prev_loss) / (np.abs(prev_loss) + np.abs(current_loss))
        if precision < 1e-6:  # good enough
            show_dots(x, y)
            display_loss(np.array(log))
            break
        #  step corection after loss increase
        if current_loss > prev_loss:
            x = x + alpha * grad[0]  # reverse step
            y = y + alpha * grad[1]
            alpha = alpha / 2


def show_dots(x, y):
    plt.axis((-15, 15, -15, 15))
    n = len(x)
    #  place red dots
    for i in range(n):
        plt.plot(x[i], y[i], 'ro')
        plt.text(x[i], y[i], f'({x[i]:.2f}, {y[i]:.2f})', size='large')
    #  connect to full graph with blue lines
    for i in range(n):
        for j in range(i+1, n):
            plt.plot((x[i], x[j]), (y[i], y[j]), 'b-', linewidth=0.5)
    plt.show()


def display_loss(log):
    plt.plot(log[:, 0], log[:, 1])
    plt.xlabel('iteration')
    plt.ylabel('loss')
    plt.show()


show_dots(x, y)
S = 10
gradient_descent(x, y, S)
